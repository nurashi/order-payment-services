package messaging

import "context"

type PaymentEvent struct {
	EventID       string  `json:"event_id"`
	OrderID       string  `json:"order_id"`
	Amount        int64   `json:"amount"`
	CustomerEmail string  `json:"customer_email"`
	Status        string  `json:"status"`
}

type EventPublisher interface {
	Publish(ctx context.Context, event *PaymentEvent) error
	Close() error
}
