package service

import (
	"fmt"
	"time"

	"github.com/nurashi/order-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentClient interface {
	ProcessPayment(orderID string, amount int64) (string, error)
}

type OrderService interface {
	CreateOrder(customerID, itemName string, amount int64) (*domain.Order, error)
	GetOrder(id string) (*domain.Order, error)
	CancelOrder(id string) error
}

type orderService struct {
	repo          domain.OrderRepository
	paymentClient PaymentClient
}

func NewOrderService(repo domain.OrderRepository, paymentClient PaymentClient) OrderService {
	return &orderService{
		repo:          repo,
		paymentClient: paymentClient,
	}
}

func (s *orderService) CreateOrder(customerID, itemName string, amount int64) (*domain.Order, error) {
	order := &domain.Order{
		ID:         uuid.New().String(),
		CustomerID: customerID,
		ItemName:   itemName,
		Amount:     amount,
		Status:     domain.OrderStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repo.Create(order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	paymentStatus, err := s.paymentClient.ProcessPayment(order.ID, amount)
	if err != nil {
		order.Status = domain.OrderStatusFailed
		order.UpdatedAt = time.Now()
		if updateErr := s.repo.Update(order); updateErr != nil {
			return nil, fmt.Errorf("failed to update order after payment error: %w", updateErr)
		}
		return nil, fmt.Errorf("payment processing failed: %w", err)
	}

	if paymentStatus == "Authorized" {
		order.Status = domain.OrderStatusPaid
	} else {
		order.Status = domain.OrderStatusFailed
	}

	order.UpdatedAt = time.Now()
	if err := s.repo.Update(order); err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	return order, nil
}

func (s *orderService) GetOrder(id string) (*domain.Order, error) {
	return s.repo.GetByID(id)
}

func (s *orderService) CancelOrder(id string) error {
	order, err := s.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order.Status != domain.OrderStatusPending {
		return fmt.Errorf("only pending orders can be cancelled, current status: %s", order.Status)
	}

	order.Status = domain.OrderStatusCancelled
	order.UpdatedAt = time.Now()

	if err := s.repo.Update(order); err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	return nil
}
