package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type TicketRepositoryMock struct {
	mock.Mock
}

func (m *TicketRepositoryMock) Create(ctx context.Context, ticket *domain.Ticket) error {
	args := m.Called(ctx, ticket)

	return args.Error(0)
}

func (m *TicketRepositoryMock) GetAvailableTickets(ctx context.Context) ([]*domain.Ticket, error) {
	args := m.Called(ctx)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*domain.Ticket), args.Error(1)
}

func (m *TicketRepositoryMock) FindByID(ctx context.Context, id string) (*domain.Ticket, error) {
	args := m.Called(ctx, id)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*domain.Ticket), args.Error(1)
}

func (m *TicketRepositoryMock) DecreaseQuota(ctx context.Context, id string, quantity int) error {
	args := m.Called(ctx, id, quantity)

	return args.Error(0)
}
