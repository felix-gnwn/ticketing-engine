package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
	"github.com/felix-gnwn/ticketing-engine/internal/usecase"
	"github.com/felix-gnwn/ticketing-engine/internal/usecase/mocks"
)

func TestCreateOrder_TableDriven(t *testing.T) {
	ctx := context.Background()

	validUserID := "usr-123"
	validTicketID := "tkt-888"
	validIdempotencyKey := "idempotency-key-abc"

	mockTicket := &domain.Ticket{
		ID:        validTicketID,
		EventName: "Coldplay Concert 2026",
		Quota:     10,
		Price:     1500000.00,
	}

	tests := []struct {
		name           string
		userID         string
		ticketID       string
		quantity       int
		idempotencyKey string
		setupMocks     func(
			orderRepo *mocks.OrderRepositoryMock,
			ticketRepo *mocks.TicketRepositoryMock,
			idempotencyRepo *mocks.IdempotencyRepositoryMock,
			eventProducer *mocks.EventProducerMock,
		)
		expectedStatus domain.OrderStatus
		expectedErr    error
	}{
		{
			name:           "Success - Create Order without Idempotency Key",
			userID:         validUserID,
			ticketID:       validTicketID,
			quantity:       2,
			idempotencyKey: "",
			setupMocks: func(
				orderRepo *mocks.OrderRepositoryMock,
				ticketRepo *mocks.TicketRepositoryMock,
				idempotencyRepo *mocks.IdempotencyRepositoryMock,
				eventProducer *mocks.EventProducerMock,
			) {
				ticketRepo.On("FindByID", mock.Anything, validTicketID).Return(mockTicket, nil).Once()
				orderRepo.On("Create", mock.Anything, mock.MatchedBy(func(o *domain.Order) bool {
					return o.UserID == validUserID && o.TicketID == validTicketID && o.Quantity == 2 && o.TotalPrice == 3000000.00
				})).Return(nil).Once()

				eventProducer.On("PublishOrderCreated", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			expectedStatus: domain.OrderStatusPending,
			expectedErr:    nil,
		},
		{
			name:           "Success - Create Order with Idempotency Key & Distributed Lock",
			userID:         validUserID,
			ticketID:       validTicketID,
			quantity:       1,
			idempotencyKey: validIdempotencyKey,
			setupMocks: func(
				orderRepo *mocks.OrderRepositoryMock,
				ticketRepo *mocks.TicketRepositoryMock,
				idempotencyRepo *mocks.IdempotencyRepositoryMock,
				eventProducer *mocks.EventProducerMock,
			) {
				idempotencyRepo.On("Lock", mock.Anything, validIdempotencyKey, 5*time.Minute).Return(true, nil).Once()
				ticketRepo.On("FindByID", mock.Anything, validTicketID).Return(mockTicket, nil).Once()
				orderRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()

				eventProducer.On("PublishOrderCreated", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			expectedStatus: domain.OrderStatusPending,
			expectedErr:    nil,
		},
		{
			name:           "Error - User ID Empty",
			userID:         "",
			ticketID:       validTicketID,
			quantity:       1,
			idempotencyKey: "",
			setupMocks: func(orderRepo *mocks.OrderRepositoryMock, ticketRepo *mocks.TicketRepositoryMock, idempotencyRepo *mocks.IdempotencyRepositoryMock, eventProducer *mocks.EventProducerMock) {
			},
			expectedErr: errors.New("user ID cannot be empty"),
		},
		{
			name:           "Error - Ticket ID Empty",
			userID:         validUserID,
			ticketID:       "",
			quantity:       1,
			idempotencyKey: "",
			setupMocks: func(orderRepo *mocks.OrderRepositoryMock, ticketRepo *mocks.TicketRepositoryMock, idempotencyRepo *mocks.IdempotencyRepositoryMock, eventProducer *mocks.EventProducerMock) {
			},
			expectedErr: domain.ErrTicketNotFound,
		},
		{
			name:           "Error - Invalid Quantity (Negative or Zero)",
			userID:         validUserID,
			ticketID:       validTicketID,
			quantity:       0,
			idempotencyKey: "",
			setupMocks: func(orderRepo *mocks.OrderRepositoryMock, ticketRepo *mocks.TicketRepositoryMock, idempotencyRepo *mocks.IdempotencyRepositoryMock, eventProducer *mocks.EventProducerMock) {
			},
			expectedErr: domain.ErrInvalidQuantity,
		},
		{
			name:           "Error - Idempotency Lock Conflict (Duplicate Order Request)",
			userID:         validUserID,
			ticketID:       validTicketID,
			quantity:       1,
			idempotencyKey: validIdempotencyKey,
			setupMocks: func(
				orderRepo *mocks.OrderRepositoryMock,
				ticketRepo *mocks.TicketRepositoryMock,
				idempotencyRepo *mocks.IdempotencyRepositoryMock,
				eventProducer *mocks.EventProducerMock,
			) {
				idempotencyRepo.On("Lock", mock.Anything, validIdempotencyKey, 5*time.Minute).Return(false, nil).Once()
			},
			expectedErr: domain.ErrDuplicateOrder,
		},
		{
			name:           "Error - Insufficient Ticket Quota (Fail Fast)",
			userID:         validUserID,
			ticketID:       validTicketID,
			quantity:       20,
			idempotencyKey: validIdempotencyKey,
			setupMocks: func(
				orderRepo *mocks.OrderRepositoryMock,
				ticketRepo *mocks.TicketRepositoryMock,
				idempotencyRepo *mocks.IdempotencyRepositoryMock,
				eventProducer *mocks.EventProducerMock,
			) {
				idempotencyRepo.On("Lock", mock.Anything, validIdempotencyKey, 5*time.Minute).Return(true, nil).Once()
				ticketRepo.On("FindByID", mock.Anything, validTicketID).Return(mockTicket, nil).Once()
				idempotencyRepo.On("Unlock", mock.Anything, validIdempotencyKey).Return(nil).Once()
			},
			expectedErr: domain.ErrInsufficientQuota,
		},
		{
			name:           "Error - Order Database Persistence Failed",
			userID:         validUserID,
			ticketID:       validTicketID,
			quantity:       1,
			idempotencyKey: validIdempotencyKey,
			setupMocks: func(
				orderRepo *mocks.OrderRepositoryMock,
				ticketRepo *mocks.TicketRepositoryMock,
				idempotencyRepo *mocks.IdempotencyRepositoryMock,
				eventProducer *mocks.EventProducerMock,
			) {
				idempotencyRepo.On("Lock", mock.Anything, validIdempotencyKey, 5*time.Minute).Return(true, nil).Once()
				ticketRepo.On("FindByID", mock.Anything, validTicketID).Return(mockTicket, nil).Once()
				orderRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db connection failure")).Once()
				idempotencyRepo.On("Unlock", mock.Anything, validIdempotencyKey).Return(nil).Once()
			},
			expectedErr: errors.New("db connection failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := new(mocks.OrderRepositoryMock)
			ticketRepo := new(mocks.TicketRepositoryMock)
			idempotencyRepo := new(mocks.IdempotencyRepositoryMock)
			eventProducer := new(mocks.EventProducerMock)

			tt.setupMocks(orderRepo, ticketRepo, idempotencyRepo, eventProducer)

			uc := usecase.NewOrderUseCase(orderRepo, ticketRepo, idempotencyRepo, eventProducer)

			result, err := uc.CreateOrder(ctx, tt.userID, tt.ticketID, tt.quantity, tt.idempotencyKey)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedStatus, result.Status)
				assert.Equal(t, tt.userID, result.UserID)
				assert.Equal(t, tt.ticketID, result.TicketID)
			}

			time.Sleep(20 * time.Millisecond)

			orderRepo.AssertExpectations(t)
			ticketRepo.AssertExpectations(t)
			idempotencyRepo.AssertExpectations(t)
		})
	}
}

func TestGetOrderByID_TableDriven(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		orderID     string
		setupMocks  func(orderRepo *mocks.OrderRepositoryMock)
		expectedErr error
	}{
		{
			name:        "Error - Empty Order ID",
			orderID:     "",
			setupMocks:  func(orderRepo *mocks.OrderRepositoryMock) {},
			expectedErr: domain.ErrOrderNotFound,
		},
		{
			name:    "Success - Order Found",
			orderID: "ORD-999",
			setupMocks: func(orderRepo *mocks.OrderRepositoryMock) {
				expectedOrder := &domain.Order{
					ID:         "ORD-999",
					UserID:     "usr-123",
					TicketID:   "tkt-888",
					Quantity:   1,
					TotalPrice: 1500000.00,
					Status:     domain.OrderStatusPending,
				}

				orderRepo.On("FindByID", mock.Anything, "ORD-999").Return(expectedOrder, nil).Once()
			}, expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := new(mocks.OrderRepositoryMock)
			tt.setupMocks(orderRepo)

			uc := usecase.NewOrderUseCase(orderRepo, nil, nil, nil)
			res, err := uc.GetOrderByID(ctx, tt.orderID)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				assert.Equal(t, tt.orderID, res.ID)
			}

			orderRepo.AssertExpectations(t)
		})
	}
}
