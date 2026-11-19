package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

type IdempotencyRepositoryMock struct {
	mock.Mock
}

func (m *IdempotencyRepositoryMock) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	args := m.Called(ctx, key, ttl)

	return args.Bool(0), args.Error(1)
}

func (m *IdempotencyRepositoryMock) Unlock(ctx context.Context, key string) error {
	args := m.Called(ctx, key)

	return args.Error(0)
}
