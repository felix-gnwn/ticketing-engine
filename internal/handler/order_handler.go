package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
	"github.com/felix-gnwn/ticketing-engine/internal/middleware"
)

type OrderHandler struct {
	orderUC domain.OrderUseCase
}

type CreateOrderRequest struct {
	TicketID string `json:"ticket_id"`
	Quantity int    `json:"quantity"`
}

type OrderResponse struct {
	ID         string             `json:"id"`
	UserID     string             `json:"user_id"`
	TicketID   string             `json:"ticket_id"`
	Quantity   int                `json:"quantity"`
	TotalPrice float64            `json:"total_price"`
	Status     domain.OrderStatus `json:"status"`
	CreatedAt  string             `json:"created_at"`
	UpdatedAt  string             `json:"updated_at"`
}

func NewOrderHandler(orderUC domain.OrderUseCase) *OrderHandler {
	return &OrderHandler{
		orderUC: orderUC,
	}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized access"})

		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON payload"})

		return
	}

	// Read optional Idempotency-Key header for duplicate protection
	idempotencyKey := r.Header.Get("X-Idempotency-Key")
	order, err := h.orderUC.CreateOrder(r.Context(), userID, req.TicketID, req.Quantity, idempotencyKey)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case errors.Is(err, domain.ErrInvalidQuantity), errors.Is(err, domain.ErrTicketNotFound):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Is(err, domain.ErrInsufficientQuota):
			w.WriteHeader(http.StatusConflict) // 409 Conflict: Insufficient stock
		case errors.Is(err, domain.ErrDuplicateOrder):
			w.WriteHeader(http.StatusConflict) // 409 Conflict: Idempotency violation
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}

		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})

		return
	}

	resp := OrderResponse{
		ID:         order.ID,
		UserID:     order.UserID,
		TicketID:   order.TicketID,
		Quantity:   order.Quantity,
		TotalPrice: order.TotalPrice,
		Status:     order.Status,
		CreatedAt:  order.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  order.UpdatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *OrderHandler) GetOrderByID(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	if orderID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "order ID is required"})

		return
	}
	order, err := h.orderUC.GetOrderByID(r.Context(), orderID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if errors.Is(err, domain.ErrOrderNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}

		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(order)
}

func (h *OrderHandler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized access"})

		return
	}

	orders, err := h.orderUC.GetUserOrders(r.Context(), userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})

		return
	}

	if orders == nil {
		orders = []*domain.Order{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(orders)
}
