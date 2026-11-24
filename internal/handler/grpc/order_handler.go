package grpc

import (
	"context"
	"errors"
	"time"

	ticketingv1 "github.com/felix-gnwn/ticketing-engine/api/proto/v1"
	"github.com/felix-gnwn/ticketing-engine/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OrderGRPCHandler struct {
	ticketingv1.UnimplementedOrderServiceServer
	orderUC domain.OrderUseCase
}

func NewOrderGRPCHandler(orderUC domain.OrderUseCase) *OrderGRPCHandler {
	return &OrderGRPCHandler{
		orderUC: orderUC,
	}
}

func (h *OrderGRPCHandler) CreateOrder(ctx context.Context, req *ticketingv1.CreateOrderRequest) (*ticketingv1.CreateOrderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request payload cannot be nil")
	}

	order, err := h.orderUC.CreateOrder(ctx, req.GetUserId(), req.GetTicketId(), int(req.GetQuantity()), req.GetIdempotencyKey())
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidQuantity):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, domain.ErrTicketNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, domain.ErrInsufficientQuota):
			return nil, status.Error(codes.ResourceExhausted, err.Error())
		case errors.Is(err, domain.ErrDuplicateOrder):
			return nil, status.Error(codes.AlreadyExists, err.Error())
		default:
			return nil, status.Errorf(codes.Internal, "failed to process order: %v", err)
		}
	}

	pbOrder := &ticketingv1.Order{
		Id:         order.ID,
		UserId:     order.UserID,
		TicketId:   order.TicketID,
		Quantity:   int32(order.Quantity),
		TotalPrice: order.TotalPrice,
		Status:     string(order.Status),
		CreatedAt:  order.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  order.UpdatedAt.Format(time.RFC3339),
	}

	return &ticketingv1.CreateOrderResponse{
		Order: pbOrder,
	}, nil
}
