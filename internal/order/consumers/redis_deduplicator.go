package consumers

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const processedEventKeyPrefix = "order:event:processed:"

type RedisEventDeduplicator struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisEventDeduplicator(client *redis.Client, ttl time.Duration) *RedisEventDeduplicator {
	return &RedisEventDeduplicator{
		client: client,
		ttl:    ttl,
	}
}

func (d *RedisEventDeduplicator) IsProcessed(ctx context.Context, eventID string) (bool, error) {
	exists, err := d.client.Exists(ctx, processedEventKey(eventID)).Result()
	if err != nil {
		return false, fmt.Errorf("check processed event in redis: %w", err)
	}

	return exists == 1, nil
}

func (d *RedisEventDeduplicator) MarkProcessed(ctx context.Context, eventID string) error {
	if err := d.client.Set(ctx, processedEventKey(eventID), "1", d.ttl).Err(); err != nil {
		return fmt.Errorf("mark processed event in redis: %w", err)
	}

	return nil
}

func processedEventKey(eventID string) string {
	return processedEventKeyPrefix + eventID
}
