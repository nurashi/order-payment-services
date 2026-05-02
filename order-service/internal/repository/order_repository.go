package repository

import (
	"context"
	"fmt"

	"github.com/nurashi/order-service/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type orderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) domain.OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(order *domain.Order) error {
	query := `
		INSERT INTO orders (id, customer_id, customer_email, item_name, amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.Exec(
		context.Background(),
		query,
		order.ID,
		order.CustomerID,
		order.CustomerEmail,
		order.ItemName,
		order.Amount,
		order.Status,
		order.CreatedAt,
		order.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

func (r *orderRepository) GetByID(id string) (*domain.Order, error) {
	query := `
		SELECT id, customer_id, customer_email, item_name, amount, status, created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	order := &domain.Order{}
	err := r.db.QueryRow(context.Background(), query, id).Scan(
		&order.ID,
		&order.CustomerID,
		&order.CustomerEmail,
		&order.ItemName,
		&order.Amount,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	return order, nil
}

func (r *orderRepository) GetAll() ([]*domain.Order, error) {
	query := `
		SELECT id, customer_id, customer_email, item_name, amount, status, created_at, updated_at
		FROM orders
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		order := &domain.Order{}
		if err := rows.Scan(
			&order.ID,
			&order.CustomerID,
			&order.CustomerEmail,
			&order.ItemName,
			&order.Amount,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (r *orderRepository) Update(order *domain.Order) error {
	query := `
		UPDATE orders
		SET customer_id = $2, customer_email = $3, item_name = $4, amount = $5, status = $6, updated_at = $7
		WHERE id = $1
	`

	result, err := r.db.Exec(
		context.Background(),
		query,
		order.ID,
		order.CustomerID,
		order.CustomerEmail,
		order.ItemName,
		order.Amount,
		order.Status,
		order.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("order not found")
	}

	_, _ = r.db.Exec(
		context.Background(),
		"SELECT pg_notify('order_updates', $1)",
		order.ID+":"+string(order.Status),
	)

	return nil
}
