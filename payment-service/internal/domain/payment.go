package domain

import "time"

type PaymentStatus string

const (
	PaymentStatusAuthorized PaymentStatus = "Authorized"
	PaymentStatusDeclined   PaymentStatus = "Declined"
)

type Payment struct {
	ID            string
	OrderID       string
	TransactionID string
	Amount        int64
	CustomerEmail string
	Status        PaymentStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type PaymentRepository interface {
	Create(payment *Payment) error
	GetByID(id string) (*Payment, error)
	GetByOrderID(orderID string) (*Payment, error)
}
