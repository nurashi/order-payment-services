package messaging

import (
	"context"

	"github.com/nurashi/notification-service/internal/domain"
)

type EventHandler interface {
	Handle(ctx context.Context, event *domain.PaymentEvent) error
}

type EventConsumer interface {
	Start(ctx context.Context, handler EventHandler) error
	Stop() error
}
