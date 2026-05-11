package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nurashi/order-service/internal/cache"
	"github.com/nurashi/order-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentClient interface {
	ProcessPayment(orderID string, amount int64, customerEmail string) (string, error)
}

type OrderService interface {
	CreateOrder(customerID, customerEmail, itemName string, amount int64) (*domain.Order, error)
	GetOrder(id string) (*domain.Order, error)
	GetAllOrders() ([]*domain.Order, error)
	CancelOrder(id string) error
}

type orderService struct {
	repo          domain.OrderRepository
	paymentClient PaymentClient
	cache         cache.OrderCache
}

func NewOrderService(repo domain.OrderRepository, paymentClient PaymentClient, c cache.OrderCache) OrderService {
	return &orderService{
		repo:          repo,
		paymentClient: paymentClient,
		cache:         c,
	}
}

func (s *orderService) CreateOrder(customerID, customerEmail, itemName string, amount int64) (*domain.Order, error) {
	order := &domain.Order{
		ID:            uuid.New().String(),
		CustomerID:    customerID,
		CustomerEmail: customerEmail,
		ItemName:      itemName,
		Amount:        amount,
		Status:        domain.OrderStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Create(order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	paymentStatus, err := s.paymentClient.ProcessPayment(order.ID, amount, customerEmail)
	if err != nil {
		order.Status = domain.OrderStatusFailed
		order.UpdatedAt = time.Now()
		if updateErr := s.repo.Update(order); updateErr != nil {
			return nil, fmt.Errorf("failed to update order after payment error: %w", updateErr)
		}
		s.invalidateCache(order.ID)
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

	s.invalidateCache(order.ID)

	return order, nil
}

func (s *orderService) GetOrder(id string) (*domain.Order, error) {
	cached, err := s.cache.Get(context.Background(), id)
	if err != nil {
		log.Printf("cache get error for order %s: %v", id, err)
	}

	if cached != nil {
		log.Printf("[CACHE HIT] order %s", id)
		return cached, nil
	}

	log.Printf("[CACHE MISS] order %s", id)

	order, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if setErr := s.cache.Set(context.Background(), order); setErr != nil {
		log.Printf("cache set error for order %s: %v", id, setErr)
	}

	return order, nil
}

func (s *orderService) GetAllOrders() ([]*domain.Order, error) {
	return s.repo.GetAll()
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

	s.invalidateCache(id)

	return nil
}

func (s *orderService) invalidateCache(id string) {
	if err := s.cache.Delete(context.Background(), id); err != nil {
		log.Printf("cache delete error for order %s: %v", id, err)
	}
}
