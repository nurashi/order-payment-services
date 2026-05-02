package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/nurashi/notification-service/internal/domain"
	"github.com/nurashi/notification-service/internal/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitMQConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
	mu      sync.Mutex
	stopCh  chan struct{}
}

func NewRabbitMQConsumer(host, port, user, password, queue string) (messaging.EventConsumer, error) {
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
		"payment_events",
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
		"payment_events",
		false,
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	err = channel.Qos(
		1,
		0,
		false,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &rabbitMQConsumer{
		conn:   conn,
		channel: channel,
		queue:  q.Name,
		stopCh: make(chan struct{}),
	}, nil
}

func (c *rabbitMQConsumer) Start(ctx context.Context, handler messaging.EventHandler) error {
	c.mu.Lock()
	if c.channel == nil {
		c.mu.Unlock()
		return fmt.Errorf("channel is nil")
	}
	c.mu.Unlock()

	msgs, err := c.channel.Consume(
		c.queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("Notification consumer started, listening on queue: %s", c.queue)

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping consumer")
			return ctx.Err()
		case <-c.stopCh:
			log.Println("Stop signal received, stopping consumer")
			return nil
		case msg, ok := <-msgs:
			if !ok {
				log.Println("Consumer channel closed")
				return nil
			}

			if err := c.processMessage(ctx, handler, msg); err != nil {
				log.Printf("Error processing message: %v", err)
				if err := msg.Nack(false, true); err != nil {
					log.Printf("Failed to nack message: %v", err)
				}
			}
		}
	}
}

func (c *rabbitMQConsumer) processMessage(ctx context.Context, handler messaging.EventHandler, msg amqp.Delivery) error {
	var event domain.PaymentEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		if err := msg.Nack(false, false); err != nil {
			log.Printf("Failed to nack message: %v", err)
		}
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	if err := handler.Handle(ctx, &event); err != nil {
		log.Printf("Handler failed: %v", err)
		if err := msg.Nack(false, true); err != nil {
			log.Printf("Failed to nack message: %v", err)
		}
		return fmt.Errorf("handler failed: %w", err)
	}

	if err := msg.Ack(false); err != nil {
		log.Printf("Failed to ack message: %v", err)
		return fmt.Errorf("failed to ack message: %w", err)
	}

	return nil
}

func (c *rabbitMQConsumer) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	close(c.stopCh)

	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			return err
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return err
		}
	}

	log.Println("Notification consumer stopped")
	return nil
}
