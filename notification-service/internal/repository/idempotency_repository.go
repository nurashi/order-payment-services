package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nurashi/notification-service/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type idempotencyRepository struct {
	db *pgxpool.Pool
}

func NewIdempotencyRepository(db *pgxpool.Pool) domain.IdempotencyStore {
	return &idempotencyRepository{db: db}
}

func (r *idempotencyRepository) IsProcessed(eventID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM processed_events WHERE event_id = $1)`

	var exists bool
	err := r.db.QueryRow(context.Background(), query, eventID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check event: %w", err)
	}

	return exists, nil
}

func (r *idempotencyRepository) MarkProcessed(eventID string) error {
	query := `
		INSERT INTO processed_events (event_id, processed_at)
		VALUES ($1, NOW())
		ON CONFLICT (event_id) DO NOTHING
	`

	_, err := r.db.Exec(context.Background(), query, eventID)
	if err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	return nil
}

func (r *idempotencyRepository) ProcessIfNotExists(eventID string) (bool, error) {
	query := `
		INSERT INTO processed_events (event_id, processed_at)
		VALUES ($1, NOW())
		ON CONFLICT (event_id) DO NOTHING
		RETURNING event_id
	`

	var returnedID string
	err := r.db.QueryRow(context.Background(), query, eventID).Scan(&returnedID)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to insert event: %w", err)
	}

	return true, nil
}
