package domain

import (
	"context"
	"time"
)

type Ticket struct {
	ID			string		`json:"id"`
	EventName	string		`json:"event_name"`
	Quota		int			`json:"quota"`
	Price		float64		`json:"price"`
	CreatedAt	time.Time	`json:"created_at"`
	UpdatedAt	time.Time	`json:"updated_at"`
}

type TicketRepository interface {
	Create(ctx context.Context, ticket *Ticket) error
	GetAvailableTickets(ctx context.Context) ([]*Ticket, error)
	FindByID(ctx context.Context, id string) (*Ticket, error)
}

type TicketUseCase interface {
	CreateEvent(ctx context.Context, ticket *Ticket) error
	BrowseAvailable(ctx context.Context) ([]*Ticket, error)
}