package grpc

import (
	"context"
	"time"

	ticketingv1 "github.com/felix-gnwn/ticketing-engine/api/proto/v1"
	"github.com/felix-gnwn/ticketing-engine/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TicketGRPCHandler struct {
	ticketingv1.UnimplementedTicketServiceServer
	ticketUC domain.TicketUseCase
}

func NewTicketGRPCHandler(ticketUC domain.TicketUseCase) *TicketGRPCHandler {
	return &TicketGRPCHandler{
		ticketUC: ticketUC,
	}
}

func (h *TicketGRPCHandler) GetTicketAvailability(ctx context.Context, req *ticketingv1.GetTicketAvailabilityRequest) (*ticketingv1.GetTicketAvailabilityResponse, error) {
	if req == nil || req.GetTicketId() == "" {
		return nil, status.Error(codes.InvalidArgument, "ticket ID is required")
	}

	tickets, err := h.ticketUC.BrowseAvailable(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch available tickets: %v", err)
	}

	var foundTicket *domain.Ticket
	for _, t := range tickets {
		if t.ID == req.GetTicketId() {
			foundTicket = t
			break
		}
	}

	if foundTicket == nil {
		return nil, status.Error(codes.NotFound, "ticket not found or currently unavailable")
	}

	pbTicket := &ticketingv1.Ticket{
		Id:        foundTicket.ID,
		EventName: foundTicket.EventName,
		Quota:     int32(foundTicket.Quota),
		Price:     foundTicket.Price,
		CreatedAt: foundTicket.CreatedAt.Format(time.RFC3339),
		UpdatedAt: foundTicket.UpdatedAt.Format(time.RFC3339),
	}

	return &ticketingv1.GetTicketAvailabilityResponse{
		Ticket: pbTicket,
	}, nil
}
