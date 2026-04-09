package domain

import "time"

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "Pending"
	OrderStatusPaid      OrderStatus = "Paid"
	OrderStatusFailed    OrderStatus = "Failed"
	OrderStatusCancelled OrderStatus = "Cancelled"
)

type Order struct {
	ID         string
	CustomerID string
	ItemName   string
	Amount     int64
	Status     OrderStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type OrderRepository interface {
	Create(order *Order) error
	GetByID(id string) (*Order, error)
	Update(order *Order) error
}
