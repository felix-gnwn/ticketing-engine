package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type OrderRepositoryMock struct {
	mock.Mock
}

func (m *OrderRepositoryMock) Create(ctx context.Context, order *domain.Order) error {
	args := m.Called(ctx, order)

	return args.Error(0)
}

func (m *OrderRepositoryMock) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*domain.Order), args.Error(1)
}

func (m *OrderRepositoryMock) FindByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*domain.Order), args.Error(1)
}

func (m *OrderRepositoryMock) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	args := m.Called(ctx, id, status)

	return args.Error(0)
}
