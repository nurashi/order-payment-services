package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nurashi/notification-service/internal/domain"
	"github.com/redis/go-redis/v9"
)

type redisIdempotencyStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisIdempotencyStore(client *redis.Client, ttlSeconds int) domain.IdempotencyStore {
	return &redisIdempotencyStore{
		client: client,
		ttl:    time.Duration(ttlSeconds) * time.Second,
	}
}

func idempotencyKey(eventID string) string {
	return fmt.Sprintf("idempotency:%s", eventID)
}

func (r *redisIdempotencyStore) IsProcessed(eventID string) (bool, error) {
	val, err := r.client.Exists(context.Background(), idempotencyKey(eventID)).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists failed: %w", err)
	}
	return val > 0, nil
}

func (r *redisIdempotencyStore) MarkProcessed(eventID string) error {
	if err := r.client.Set(context.Background(), idempotencyKey(eventID), "1", r.ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}
	return nil
}

func (r *redisIdempotencyStore) ProcessIfNotExists(eventID string) (bool, error) {
	set, err := r.client.SetNX(context.Background(), idempotencyKey(eventID), "1", r.ttl).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return false, fmt.Errorf("redis setnx failed: %w", err)
	}
	return set, nil
}
