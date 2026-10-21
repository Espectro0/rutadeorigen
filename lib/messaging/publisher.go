package messaging

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const Exchange = "rutadeorigen.events"

type Message struct {
	ID, Type, CorrelationID string
	Version                 int
	OccurredAt              time.Time
	Body                    []byte
}

type Publisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewPublisher(url string) (*Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("open rabbitmq channel: %w", err), conn.Close())
	}
	if err := ch.Confirm(false); err != nil {
		return nil, errors.Join(fmt.Errorf("enable publisher confirms: %w", err), conn.Close())
	}
	if err := ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		return nil, errors.Join(fmt.Errorf("declare exchange %s: %w", Exchange, err), conn.Close())
	}
	return &Publisher{conn: conn, ch: ch}, nil
}

func (p *Publisher) Publish(ctx context.Context, m Message) error {
	conf, err := p.ch.PublishWithDeferredConfirmWithContext(ctx, Exchange, m.Type, false, false,
		amqp.Publishing{
			MessageId:     m.ID,
			Type:          m.Type,
			CorrelationId: m.CorrelationID,
			Timestamp:     m.OccurredAt,
			ContentType:   "application/json",
			DeliveryMode:  amqp.Persistent,
			Headers:       amqp.Table{"version": int64(m.Version)},
			Body:          m.Body,
		})
	if err != nil {
		return fmt.Errorf("publish %s: %w", m.Type, err)
	}
	ok, err := conf.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("wait confirm for %s: %w", m.Type, err)
	}
	if !ok {
		return errors.New("message nacked by broker")
	}
	return nil
}

func (p *Publisher) Ping(context.Context) error {
	if p.conn.IsClosed() {
		return errors.New("rabbitmq connection closed")
	}
	return nil
}

func (p *Publisher) Closed() <-chan *amqp.Error {
	return p.conn.NotifyClose(make(chan *amqp.Error, 1))
}

func (p *Publisher) Close(context.Context) error {
	if err := p.conn.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
		return fmt.Errorf("close rabbitmq connection: %w", err)
	}
	return nil
}
