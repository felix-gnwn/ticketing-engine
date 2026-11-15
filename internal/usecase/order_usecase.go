package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type orderUseCase struct {
	orderRepo       domain.OrderRepository
	ticketRepo      domain.TicketRepository
	idempotencyRepo domain.IdempotencyRepository
	eventProducer   domain.EventProducer
}

func NewOrderUseCase(orderRepo domain.OrderRepository, ticketRepo domain.TicketRepository, idempotencyRepo domain.IdempotencyRepository, eventProducer domain.EventProducer) domain.OrderUseCase {
	return &orderUseCase{
		orderRepo:       orderRepo,
		ticketRepo:      ticketRepo,
		idempotencyRepo: idempotencyRepo,
		eventProducer:   eventProducer,
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

	// Redis Idempotency Gatekeeper (Distributed Atomic Lock SETNX + TTL 5m)
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

	if u.eventProducer != nil {
		event := domain.OrderCreatedEvent{
			OrderID:    order.ID,
			UserID:     order.UserID,
			TicketID:   order.TicketID,
			Quantity:   order.Quantity,
			TotalPrice: order.TotalPrice,
			CreatedAt:  order.CreatedAt,
		}

		// Dispatch via background goroutine, so that HTTP response latency is not burdened
		go func() {
			asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if pubErr := u.eventProducer.PublishOrderCreated(asyncCtx, event); pubErr != nil {
				slog.Error("Background Event Producer failed to dispatch OrderCreated event",
					slog.String("order_id", order.ID),
					slog.String("error", pubErr.Error()),
				)
			}
		}()
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
