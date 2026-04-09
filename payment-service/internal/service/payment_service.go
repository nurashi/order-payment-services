package service

import (
	"fmt"
	"time"

	"github.com/nurashi/payment-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentService interface {
	ProcessPayment(orderID string, amount int64) (*domain.Payment, error)
	GetPayment(id string) (*domain.Payment, error)
	GetPaymentByOrderID(orderID string) (*domain.Payment, error)
}

type paymentService struct {
	repo domain.PaymentRepository
}

func NewPaymentService(repo domain.PaymentRepository) PaymentService {
	return &paymentService{repo: repo}
}

func (s *paymentService) ProcessPayment(orderID string, amount int64) (*domain.Payment, error) {
	status := domain.PaymentStatusAuthorized
	if amount > 100000 {
		status = domain.PaymentStatusDeclined
	}

	payment := &domain.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		TransactionID: uuid.New().String(),
		Amount:        amount,
		Status:        status,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := s.repo.Create(payment)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	return payment, nil
}

func (s *paymentService) GetPayment(id string) (*domain.Payment, error) {
	return s.repo.GetByID(id)
}

func (s *paymentService) GetPaymentByOrderID(orderID string) (*domain.Payment, error) {
	return s.repo.GetByOrderID(orderID)
}
