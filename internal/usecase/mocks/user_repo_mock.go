package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type UserRepositoryMock struct {
	mock.Mock
}

func (m *UserRepositoryMock) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)

	return args.Error(0)
}

func (m *UserRepositoryMock) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)

	if args.Get(0) == nil {
		return nil, args.Error(0)
	}

	return args.Get(0).(*domain.User), args.Error(1)
}
