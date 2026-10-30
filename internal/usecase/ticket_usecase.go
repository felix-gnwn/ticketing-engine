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

func (uc *ticketUseCase) CreateEvent(ctx context.Context, ticket *domain.Ticket) error {
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

func (uc *ticketUseCase) CheckoutTicket(ctx context.Context, ticketID string, quantity int) error {
	ticket, err := uc.ticketRepo.FindByID(ctx, ticketID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch ticket for checkout",
			slog.String("ticket_id", ticketID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("repository failed to fetch ticket: %w", err)
	}

	if ticket.Quota < quantity {
		slog.WarnContext(ctx, "checkout rejected: insufficient quota",
			slog.String("ticket_id", ticketID),
			slog.Int("current_quota", ticket.Quota),
			slog.Int("requested_quantity", quantity),
		)
		return domain.ErrTicketUnavailable
	}

	// Anti race condition handler will be in repository layer
	err = uc.ticketRepo.DecreaseQuota(ctx, ticketID, quantity)
	if err != nil {
		slog.ErrorContext(ctx, "failed to decrease quota during checkout",
			slog.String("ticket_id", ticketID),
			slog.Int("quantity", quantity),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("repository failed to decrease quota: %w", err)
	}

	slog.InfoContext(ctx, "ticket successfully checked out",
		slog.String("ticket_id", ticketID),
		slog.Int("quantity", quantity),
	)

	return nil
}