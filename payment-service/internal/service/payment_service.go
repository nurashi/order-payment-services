package service

import (
	"context"
	"fmt"
	"time"

	"github.com/nurashi/payment-service/internal/domain"
	"github.com/nurashi/payment-service/internal/messaging"

	"github.com/google/uuid"
)

type PaymentService interface {
	ProcessPayment(orderID string, amount int64, customerEmail string) (*domain.Payment, error)
	GetPayment(id string) (*domain.Payment, error)
	GetPaymentByOrderID(orderID string) (*domain.Payment, error)
}

type paymentService struct {
	repo      domain.PaymentRepository
	publisher messaging.EventPublisher
}

func NewPaymentService(repo domain.PaymentRepository, publisher messaging.EventPublisher) PaymentService {
	return &paymentService{repo: repo, publisher: publisher}
}

func (s *paymentService) ProcessPayment(orderID string, amount int64, customerEmail string) (*domain.Payment, error) {
	status := domain.PaymentStatusAuthorized
	if amount > 100000 {
		status = domain.PaymentStatusDeclined
	}

	payment := &domain.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		TransactionID: uuid.New().String(),
		Amount:        amount,
		CustomerEmail: customerEmail,
		Status:        status,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := s.repo.Create(payment)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	event := &messaging.PaymentEvent{
		EventID:       uuid.New().String(),
		OrderID:       payment.OrderID,
		Amount:        payment.Amount,
		CustomerEmail: payment.CustomerEmail,
		Status:        string(payment.Status),
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(context.Background(), event); err != nil {
			return payment, fmt.Errorf("payment created but failed to publish event: %w", err)
		}
	}

	return payment, nil
}

func (s *paymentService) GetPayment(id string) (*domain.Payment, error) {
	return s.repo.GetByID(id)
}

func (s *paymentService) GetPaymentByOrderID(orderID string) (*domain.Payment, error) {
	return s.repo.GetByOrderID(orderID)
}
