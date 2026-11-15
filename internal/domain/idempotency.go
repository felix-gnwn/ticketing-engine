package domain

import (
	"context"
	"time"
)

type IdempotencyRepository interface {
	// Lock attempts to acquire an atomic lock for a given idempotency key.
	// Returns true if key was successfully acquired, or false if key already exists.
	Lock(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// Unlock releases an idempotency key lock prematurely if an operation fails.
	Unlock(ctx context.Context, key string) error
}
