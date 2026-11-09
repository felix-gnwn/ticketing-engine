package domain

import (
	"context"
	"time"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusPaid      OrderStatus = "PAID"
	OrderStatusCancelled OrderStatus = "CANCELLED"
	OrderStatusExpired   OrderStatus = "EXPIRED"
)

type Order struct {
	ID         string      `json:"id"`
	UserID     string      `json:"user_id"`
	TicketID   string      `json:"ticket_id"`
	Quantity   int         `json:"quantity"`
	TotalPrice float64     `json:"total_price"`
	Status     OrderStatus `json:"status"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

type OrderRepository interface {
	Create(ctx context.Context, order *Order) error
	FindByID(ctx context.Context, id string) (*Order, error)
	FindByUserID(ctx context.Context, userID string) ([]*Order, error)
	UpdateStatus(ctx context.Context, id string, status OrderStatus) error
}

type OrderUseCase interface {
	CreateOrder(ctx context.Context, userID, ticketID string, quantity int, idempotencyKey string) (*Order, error)
	GetOrderByID(ctx context.Context, id string) (*Order, error)
	GetUserOrders(ctx context.Context, userID string) ([]*Order, error)
}
