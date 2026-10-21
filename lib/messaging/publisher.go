package messaging

import (
	"context"
	"errors"
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
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := ch.Confirm(false); err != nil {
		conn.Close()
		return nil, err
	}
	if err := ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		conn.Close()
		return nil, err
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
			Headers:       amqp.Table{"version": int32(m.Version)},
			Body:          m.Body,
		})
	if err != nil {
		return err
	}
	ok, err := conf.WaitContext(ctx)
	if err != nil {
		return err
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

func (p *Publisher) Close(context.Context) error { return p.conn.Close() }
