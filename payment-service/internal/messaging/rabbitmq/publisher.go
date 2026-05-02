package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nurashi/payment-service/internal/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	exchange string
	queue   string
}

func NewRabbitMQPublisher(host, port, user, password, exchange, queue string) (messaging.EventPublisher, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", user, password, host, port)

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	err = channel.ExchangeDeclare(
		exchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	q, err := channel.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	err = channel.QueueBind(
		q.Name,
		queue,
		exchange,
		false,
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	return &rabbitMQPublisher{
		conn:     conn,
		channel:  channel,
		exchange: exchange,
		queue:    queue,
	}, nil
}

func (p *rabbitMQPublisher) Publish(ctx context.Context, event *messaging.PaymentEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         body,
	}

	err = p.channel.PublishWithContext(
		ctx,
		p.exchange,
		p.queue,
		false,
		false,
		msg,
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (p *rabbitMQPublisher) Close() error {
	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			return err
		}
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			return err
		}
	}
	return nil
}
