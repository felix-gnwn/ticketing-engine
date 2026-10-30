package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type ticketUseCase struct {
	ticketRepo domain.TicketRepository
}

func NewTicketUseCase(tr domain.TicketRepository) domain.TicketUseCase {
	return &ticketUseCase{
		ticketRepo: tr,
	}
}

func (uc *ticketUseCase) CheckoutTicket(ctx context.Context, userID, eventID string, quantity int) (*domain.Ticket, error) {
	if ticket.EventName == "" {
		return domain.ErrInvalidEventName
	}

	if ticket.Quota <= 0 {
		return domain.ErrInvalidQuota
	}

	if ticket.Price < 0 {
		return domain.ErrInvalidPrice
	}

	// Data Preparation (Set Audit Timestamps)
	now := time.Now().UTC()
	ticket.CreatedAt = now
	ticket.UpdatedAt = now

	err := uc.ticketRepo.Create(ctx, ticket)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create event ticket",
			slog.String("event_name", ticket.EventName),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("repository failed to create ticket: %w", err)
	}

	slog.InfoContext(ctx, "event ticket created successfully",
		slog.String("event_name", ticket.EventName),
		slog.Int("quota", ticket.Quota),
	)

	return nil
}

func (uc *ticketUseCase) BrowseAvailable(ctx context.Context) ([]*domain.Ticket, error) {
	tickets, err := uc.ticketRepo.GetAvailableTickets(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch available tickets",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("repository failed to fetch tickets: %w", err)
	}

	slog.InfoContext(ctx, "available tickets fetched",
		slog.Int("count", len(tickets)),
	)

	return tickets, nil
}