package messaging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"rutadeorigen/lib/requestcontext"
)

const (
	RetryExchange      = "rutadeorigen.retry"
	DeadLetterExchange = "rutadeorigen.dlx"

	attemptsHeader  = "x-attempts"
	lastErrorHeader = "x-last-error"
	maxErrorLength  = 512

	defaultPrefetch    = 10
	defaultMaxAttempts = 5
	defaultRetryDelay  = 10 * time.Second
)

type Delivery struct {
	ID, Type, CorrelationID string
	Version                 int
	OccurredAt              time.Time
	Attempt                 int
	Body                    []byte
}

type Handler func(ctx context.Context, d Delivery) error

type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanentError{err: err}
}

func IsPermanent(err error) bool {
	var p permanentError
	return errors.As(err, &p)
}

type ConsumerConfig struct {
	URL         string
	Queue       string
	Bindings    []string
	Prefetch    int
	MaxAttempts int
	RetryDelay  time.Duration
}

func (c ConsumerConfig) withDefaults() ConsumerConfig {
	if c.Prefetch <= 0 {
		c.Prefetch = defaultPrefetch
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = defaultMaxAttempts
	}
	if c.RetryDelay <= 0 {
		c.RetryDelay = defaultRetryDelay
	}
	return c
}

type Consumer struct {
	cfg   ConsumerConfig
	log   *slog.Logger
	conn  *amqp.Connection
	ch    *amqp.Channel
	retry *amqp.Channel
}

func NewConsumer(cfg ConsumerConfig, log *slog.Logger) (*Consumer, error) {
	cfg = cfg.withDefaults()
	if cfg.Queue == "" || len(cfg.Bindings) == 0 {
		return nil, errors.New("consumer needs a queue and at least one binding")
	}

	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	c := &Consumer{cfg: cfg, log: log, conn: conn}
	if err := c.setup(); err != nil {
		return nil, errors.Join(err, conn.Close())
	}
	return c, nil
}

func (c *Consumer) setup() error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open consumer channel: %w", err)
	}
	c.ch = ch

	retry, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open retry channel: %w", err)
	}
	if err := retry.Confirm(false); err != nil {
		return fmt.Errorf("enable retry confirms: %w", err)
	}
	c.retry = retry

	if err := c.declareTopology(); err != nil {
		return err
	}
	if err := ch.Qos(c.cfg.Prefetch, 0, false); err != nil {
		return fmt.Errorf("set prefetch: %w", err)
	}
	return nil
}

func (c *Consumer) declareTopology() error {
	q := c.cfg.Queue
	retryQueue, deadQueue := q+".retry", q+".dlq"

	exchanges := []struct{ name, kind string }{
		{Exchange, "topic"},
		{RetryExchange, "direct"},
		{DeadLetterExchange, "direct"},
	}
	for _, e := range exchanges {
		if err := c.ch.ExchangeDeclare(e.name, e.kind, true, false, false, false, nil); err != nil {
			return fmt.Errorf("declare exchange %s: %w", e.name, err)
		}
	}

	queues := []struct {
		name string
		args amqp.Table
	}{
		{q, amqp.Table{
			"x-dead-letter-exchange":    DeadLetterExchange,
			"x-dead-letter-routing-key": q,
		}},
		{retryQueue, amqp.Table{
			"x-message-ttl":             c.cfg.RetryDelay.Milliseconds(),
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": q,
		}},
		{deadQueue, nil},
	}
	for _, d := range queues {
		if _, err := c.ch.QueueDeclare(d.name, true, false, false, false, d.args); err != nil {
			return fmt.Errorf("declare queue %s: %w", d.name, err)
		}
	}

	bindings := []struct{ queue, key, exchange string }{
		{retryQueue, q, RetryExchange},
		{deadQueue, q, DeadLetterExchange},
	}
	for _, key := range c.cfg.Bindings {
		bindings = append(bindings, struct{ queue, key, exchange string }{q, key, Exchange})
	}
	for _, b := range bindings {
		if err := c.ch.QueueBind(b.queue, b.key, b.exchange, false, nil); err != nil {
			return fmt.Errorf("bind %s to %s with %q: %w", b.queue, b.exchange, b.key, err)
		}
	}
	return nil
}

func (c *Consumer) Run(ctx context.Context, h Handler) error {
	tag := c.cfg.Queue + "-consumer"
	deliveries, err := c.ch.ConsumeWithContext(ctx, c.cfg.Queue, tag, false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume %s: %w", c.cfg.Queue, err)
	}

	for {
		select {
		case <-ctx.Done():
			if err := c.ch.Cancel(tag, false); err != nil && !errors.Is(err, amqp.ErrClosed) {
				return fmt.Errorf("cancel consumer: %w", err)
			}
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("delivery channel closed")
			}
			c.handle(ctx, d, h)
		}
	}
}

func (c *Consumer) handle(ctx context.Context, d amqp.Delivery, h Handler) {
	msg := toDelivery(d)
	ctx = requestcontext.WithCorrelationID(ctx, msg.CorrelationID)
	log := c.log.With("message_id", msg.ID, "event_type", msg.Type, "attempt", msg.Attempt)

	if msg.ID == "" {
		log.ErrorContext(ctx, "message without id sent to the dead letter queue")
		c.reject(ctx, d, log)
		return
	}

	err := safeHandle(ctx, h, msg)
	if err == nil {
		if ackErr := d.Ack(false); ackErr != nil {
			log.ErrorContext(ctx, "ack failed", "err", ackErr)
		}
		return
	}

	if ctx.Err() != nil {
		log.WarnContext(ctx, "shutting down, message returned to the queue", "err", err)
		if nackErr := d.Nack(false, true); nackErr != nil {
			log.ErrorContext(ctx, "requeue failed", "err", nackErr)
		}
		return
	}

	if IsPermanent(err) || msg.Attempt >= c.cfg.MaxAttempts {
		log.ErrorContext(ctx, "message sent to the dead letter queue", "err", err)
		c.reject(ctx, d, log)
		return
	}

	if retryErr := c.scheduleRetry(ctx, d, msg.Attempt, err); retryErr != nil {
		log.ErrorContext(ctx, "retry could not be scheduled, message sent to the dead letter queue",
			"err", err, "retry_err", retryErr)
		c.reject(ctx, d, log)
		return
	}

	log.WarnContext(ctx, "message scheduled for retry", "err", err, "delay", c.cfg.RetryDelay)
	if ackErr := d.Ack(false); ackErr != nil {
		log.ErrorContext(ctx, "ack after retry failed", "err", ackErr)
	}
}

func safeHandle(ctx context.Context, h Handler, d Delivery) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handler panic: %v", r)
		}
	}()
	return h(ctx, d)
}

func (c *Consumer) reject(ctx context.Context, d amqp.Delivery, log *slog.Logger) {
	if err := d.Nack(false, false); err != nil {
		log.ErrorContext(ctx, "nack failed", "err", err)
	}
}

func (c *Consumer) scheduleRetry(ctx context.Context, d amqp.Delivery, attempt int, cause error) error {
	headers := amqp.Table{}
	for k, v := range d.Headers {
		headers[k] = v
	}
	headers[attemptsHeader] = int64(attempt)
	headers[lastErrorHeader] = truncate(cause.Error(), maxErrorLength)

	conf, err := c.retry.PublishWithDeferredConfirmWithContext(ctx, RetryExchange, c.cfg.Queue, false, false,
		amqp.Publishing{
			MessageId:     d.MessageId,
			Type:          d.Type,
			CorrelationId: d.CorrelationId,
			Timestamp:     d.Timestamp,
			ContentType:   d.ContentType,
			DeliveryMode:  amqp.Persistent,
			Headers:       headers,
			Body:          d.Body,
		})
	if err != nil {
		return fmt.Errorf("publish retry: %w", err)
	}
	ok, err := conf.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("wait retry confirm: %w", err)
	}
	if !ok {
		return errors.New("retry nacked by broker")
	}
	return nil
}

func toDelivery(d amqp.Delivery) Delivery {
	return Delivery{
		ID:            d.MessageId,
		Type:          d.Type,
		CorrelationID: d.CorrelationId,
		Version:       headerInt(d.Headers, "version"),
		OccurredAt:    d.Timestamp,
		Attempt:       headerInt(d.Headers, attemptsHeader) + 1,
		Body:          d.Body,
	}
}

func headerInt(h amqp.Table, key string) int {
	switch v := h[key].(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		if v < 0 || v > math.MaxInt32 {
			return 0
		}
		return int(v)
	default:
		return 0
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func (c *Consumer) Ping(context.Context) error {
	if c.conn.IsClosed() {
		return errors.New("rabbitmq connection closed")
	}
	return nil
}

func (c *Consumer) Closed() <-chan *amqp.Error {
	return c.conn.NotifyClose(make(chan *amqp.Error, 1))
}

func (c *Consumer) Close(context.Context) error {
	if err := c.conn.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
		return fmt.Errorf("close rabbitmq connection: %w", err)
	}
	return nil
}
