package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nurashi/order-service/internal/domain"
	"github.com/redis/go-redis/v9"
)

type OrderCache interface {
	Get(ctx context.Context, id string) (*domain.Order, error)
	Set(ctx context.Context, order *domain.Order) error
	Delete(ctx context.Context, id string) error
}

type redisOrderCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisOrderCache(client *redis.Client, ttlSeconds int) OrderCache {
	return &redisOrderCache{
		client: client,
		ttl:    time.Duration(ttlSeconds) * time.Second,
	}
}

func cacheKey(id string) string {
	return fmt.Sprintf("order:%s", id)
}

func (c *redisOrderCache) Get(ctx context.Context, id string) (*domain.Order, error) {
	val, err := c.client.Get(ctx, cacheKey(id)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var order domain.Order
	if err := json.Unmarshal([]byte(val), &order); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached order: %w", err)
	}

	return &order, nil
}

func (c *redisOrderCache) Set(ctx context.Context, order *domain.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	if err := c.client.Set(ctx, cacheKey(order.ID), data, c.ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	return nil
}

func (c *redisOrderCache) Delete(ctx context.Context, id string) error {
	if err := c.client.Del(ctx, cacheKey(id)).Err(); err != nil {
		return fmt.Errorf("redis delete failed: %w", err)
	}
	return nil
}
