package domain

type PaymentEvent struct {
	EventID       string `json:"event_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

type IdempotencyStore interface {
	IsProcessed(eventID string) (bool, error)
	MarkProcessed(eventID string) error
	ProcessIfNotExists(eventID string) (bool, error)
}
