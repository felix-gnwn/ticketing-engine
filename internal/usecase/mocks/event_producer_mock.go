package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type EventProducerMock struct {
	mock.Mock
}

func (m *EventProducerMock) PublishOrderCreated(ctx context.Context, event domain.OrderCreatedEvent) error {
	args := m.Called(ctx, event)

	return args.Error(0)
}

func (m *EventProducerMock) Close() error {
	args := m.Called()

	return args.Error(0)
}
