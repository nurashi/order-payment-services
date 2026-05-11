package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nurashi/notification-service/internal/domain"
	"github.com/nurashi/notification-service/internal/provider"
)

type NotificationService struct {
	idempotencyStore      domain.IdempotencyStore
	sender                provider.EmailSender
	maxAttempts           int
	initialBackoffSeconds int
}

func NewNotificationService(
	idempotencyStore domain.IdempotencyStore,
	sender provider.EmailSender,
	maxAttempts int,
	initialBackoffSeconds int,
) *NotificationService {
	return &NotificationService{
		idempotencyStore:      idempotencyStore,
		sender:                sender,
		maxAttempts:           maxAttempts,
		initialBackoffSeconds: initialBackoffSeconds,
	}
}

func (s *NotificationService) Handle(ctx context.Context, event *domain.PaymentEvent) error {
	processed, err := s.idempotencyStore.ProcessIfNotExists(event.EventID)
	if err != nil {
		return fmt.Errorf("failed to check idempotency: %w", err)
	}

	if !processed {
		log.Printf("[NOTIFICATION] duplicate event, skipping: %s", event.EventID)
		return nil
	}

	subject := fmt.Sprintf("Payment confirmation for order %s", event.OrderID)
	body := fmt.Sprintf(
		"Your payment of $%.2f for order %s has been processed with status: %s",
		float64(event.Amount)/100.0,
		event.OrderID,
		event.Status,
	)

	var sendErr error
	for attempt := 0; attempt < s.maxAttempts; attempt++ {
		sendErr = s.sender.Send(event.CustomerEmail, subject, body)
		if sendErr == nil {
			log.Printf("[NOTIFICATION] sent to %s for order %s", event.CustomerEmail, event.OrderID)
			return nil
		}

		backoff := time.Duration(s.initialBackoffSeconds*(1<<attempt)) * time.Second
		log.Printf("[NOTIFICATION] attempt %d failed for event %s: %v; retrying in %s",
			attempt+1, event.EventID, sendErr, backoff)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}

	return fmt.Errorf("all %d attempts failed for event %s: %w", s.maxAttempts, event.EventID, sendErr)
}
