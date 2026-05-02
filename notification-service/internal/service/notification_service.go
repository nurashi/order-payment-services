package service

import (
	"context"
	"fmt"
	"log"

	"github.com/nurashi/notification-service/internal/domain"
)

type NotificationService struct {
	idempotencyStore domain.IdempotencyStore
}

func NewNotificationService(idempotencyStore domain.IdempotencyStore) *NotificationService {
	return &NotificationService{idempotencyStore: idempotencyStore}
}

func (s *NotificationService) Handle(ctx context.Context, event *domain.PaymentEvent) error {
	processed, err := s.idempotencyStore.ProcessIfNotExists(event.EventID)
	if err != nil {
		return fmt.Errorf("failed to check idempotency: %w", err)
	}

	if !processed {
		log.Printf("Duplicate event detected, skipping: %s", event.EventID)
		return nil
	}

	log.Printf("[Notification] Sent email to %s for Order #%s. Amount: $%.2f",
		event.CustomerEmail,
		event.OrderID,
		float64(event.Amount)/100.0,
	)

	return nil
}
