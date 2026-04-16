package event

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// StreamPublisher publishes fields into a stream transport.
type StreamPublisher interface {
	Publish(ctx context.Context, stream string, values map[string]any) error
	Ping(ctx context.Context) error
}

// RedisStreamPublisher publishes events to Redis Streams.
type RedisStreamPublisher struct {
	client *redis.Client
}

// NewRedisStreamPublisher constructs a Redis stream publisher.
func NewRedisStreamPublisher(client *redis.Client) *RedisStreamPublisher {
	return &RedisStreamPublisher{client: client}
}

// Publish appends an event to the target stream.
func (p *RedisStreamPublisher) Publish(ctx context.Context, stream string, values map[string]any) error {
	if err := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Err(); err != nil {
		return fmt.Errorf("xadd stream %s: %w", stream, err)
	}

	return nil
}

// Ping checks publisher connectivity.
func (p *RedisStreamPublisher) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}
