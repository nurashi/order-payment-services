package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nurashi/order-service/internal/domain"
)

type orderSubscriber struct {
	dsn  string
	repo domain.OrderRepository
}

func NewOrderSubscriber(dsn string, repo domain.OrderRepository) domain.OrderSubscriber {
	return &orderSubscriber{dsn: dsn, repo: repo}
}

func (s *orderSubscriber) SubscribeToOrderUpdates(ctx context.Context, orderID string) (<-chan *domain.Order, error) {
	conn, err := pgxpool.New(ctx, s.dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect for subscription: %w", err)
	}

	raw, err := conn.Acquire(ctx)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}

	_, err = raw.Exec(ctx, "LISTEN order_updates")
	if err != nil {
		raw.Release()
		conn.Close()
		return nil, fmt.Errorf("failed to LISTEN: %w", err)
	}

	ch := make(chan *domain.Order, 8)

	go func() {
		defer close(ch)
		defer raw.Release()
		defer conn.Close()

		for {
			notification, err := raw.Conn().WaitForNotification(ctx)
			if err != nil {
				return
			}

			parts := strings.SplitN(notification.Payload, ":", 2)
			if len(parts) != 2 || parts[0] != orderID {
				continue
			}

			order, err := s.repo.GetByID(parts[0])
			if err != nil {
				continue
			}
			ch <- order
		}
	}()

	return ch, nil
}
