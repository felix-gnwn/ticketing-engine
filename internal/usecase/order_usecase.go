package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type orderUseCase struct {
	orderRepo       domain.OrderRepository
	ticketRepo      domain.TicketRepository
	idempotencyRepo domain.IdempotencyRepository
}

func NewOrderUseCase(orderRepo domain.OrderRepository, ticketRepo domain.TicketRepository, idempotencyRepo domain.IdempotencyRepository) domain.OrderUseCase {
	return &orderUseCase{
		orderRepo:       orderRepo,
		ticketRepo:      ticketRepo,
		idempotencyRepo: idempotencyRepo,
	}
}

func (u *orderUseCase) CreateOrder(ctx context.Context, userID, ticketID string, quantity int, idempotencyKey string) (*domain.Order, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	if ticketID == "" {
		return nil, domain.ErrTicketNotFound
	}

	if quantity <= 0 {
		return nil, domain.ErrInvalidQuantity
	}

	if idempotencyKey != "" {
		acquired, err := u.idempotencyRepo.Lock(ctx, idempotencyKey, 5*time.Minute)
		if err != nil {
			return nil, fmt.Errorf("failed to process idempotency lock: %w", err)
		}

		if !acquired {
			return nil, domain.ErrDuplicateOrder
		}
	}

	ticket, err := u.ticketRepo.FindByID(ctx, ticketID)
	if err != nil {
		if idempotencyKey != "" {
			_ = u.idempotencyRepo.Unlock(ctx, idempotencyKey)
		}

		return nil, err
	}

	// Early Fail-Fast Check: Validate physical stock availability before opening DB transaction
	if ticket.Quota < quantity {
		if idempotencyKey != "" {
			_ = u.idempotencyRepo.Unlock(ctx, idempotencyKey)
		}

		return nil, domain.ErrInsufficientQuota
	}

	totalPrice := ticket.Price * float64(quantity)
	now := time.Now().UTC()
	orderID := fmt.Sprintf("ORD-%d", now.UnixNano())

	order := &domain.Order{
		ID:         orderID,
		UserID:     userID,
		TicketID:   ticketID,
		Quantity:   quantity,
		TotalPrice: totalPrice,
		Status:     domain.OrderStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := u.orderRepo.Create(ctx, order); err != nil {
		if idempotencyKey != "" {
			_ = u.idempotencyRepo.Unlock(ctx, idempotencyKey)
		}

		return nil, err
	}

	return order, nil
}

func (u *orderUseCase) GetOrderByID(ctx context.Context, id string) (*domain.Order, error) {
	if id == "" {
		return nil, domain.ErrOrderNotFound
	}

	return u.orderRepo.FindByID(ctx, id)
}

func (u *orderUseCase) GetUserOrders(ctx context.Context, userID string) ([]*domain.Order, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	return u.orderRepo.FindByUserID(ctx, userID)
}
