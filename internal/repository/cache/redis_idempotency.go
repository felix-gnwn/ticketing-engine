package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type redisIdempotencyRepo struct {
	client *redis.Client
}

func NewRedisIdempotencyRepository(client *redis.Client) domain.IdempotencyRepository {
	return &redisIdempotencyRepo{
		client: client,
	}
}

func (r *redisIdempotencyRepo) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	fullKey := fmt.Sprintf("idempotency:%s", key)
	success, err := r.client.SetNX(ctx, fullKey, "LOCKED", ttl).Result()
	if err != nil {
		return false, err
	}

	return success, nil
}

func (r *redisIdempotencyRepo) Unlock(ctx context.Context, key string) error {
	fullKey := fmt.Sprintf("idempotency:%s", key)

	return r.client.Del(ctx, fullKey).Err()
}
