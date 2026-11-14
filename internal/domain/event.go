package domain

import (
	"context"
	"time"
)

type OrderCreatedEvent struct {
	OrderID    string    `json:"order_id"`
	UserID     string    `json:"user_id"`
	TicketID   string    `json:"ticket_id"`
	Quantity   int       `json:"quantity"`
	TotalPrice float64   `json:"total_price"`
	CreatedAt  time.Time `json:"created_at"`
}

type EventProducer interface {
	PublishOrderCreated(ctx context.Context, event OrderCreatedEvent) error
	Close() error
}
