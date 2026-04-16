package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// BalanceCache provides wallet balance cache operations.
type BalanceCache interface {
	Get(ctx context.Context, walletID uuid.UUID) (int64, bool, error)
	Set(ctx context.Context, walletID uuid.UUID, balance int64) error
	Delete(ctx context.Context, walletID uuid.UUID) error
	Ping(ctx context.Context) error
}

// RedisBalanceCache stores balances in Redis.
type RedisBalanceCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisClient creates the shared Redis client.
func NewRedisClient(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

// NewRedisBalanceCache constructs a Redis-backed balance cache.
func NewRedisBalanceCache(client *redis.Client, ttl time.Duration) *RedisBalanceCache {
	return &RedisBalanceCache{
		client: client,
		ttl:    ttl,
	}
}

// Ping checks cache availability.
func (c *RedisBalanceCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Get reads a cached wallet balance.
func (c *RedisBalanceCache) Get(ctx context.Context, walletID uuid.UUID) (int64, bool, error) {
	value, err := c.client.Get(ctx, balanceKey(walletID)).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("get balance cache: %w", err)
	}

	balance, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parse balance cache: %w", err)
	}

	return balance, true, nil
}

// Set writes a cached wallet balance.
func (c *RedisBalanceCache) Set(ctx context.Context, walletID uuid.UUID, balance int64) error {
	return c.client.Set(ctx, balanceKey(walletID), strconv.FormatInt(balance, 10), c.ttl).Err()
}

// Delete evicts a cached wallet balance.
func (c *RedisBalanceCache) Delete(ctx context.Context, walletID uuid.UUID) error {
	return c.client.Del(ctx, balanceKey(walletID)).Err()
}

func balanceKey(walletID uuid.UUID) string {
	return "wallet:balance:" + walletID.String()
}
