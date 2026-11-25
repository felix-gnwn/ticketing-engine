package handler

import (
	"encoding/json"
	"net/http"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type TicketHandler struct {
	ticketUC domain.TicketUseCase
}

func NewTicketHandler(ticketUC domain.TicketUseCase) *TicketHandler {
	return &TicketHandler{
		ticketUC: ticketUC,
	}
}

// Request & Response Payloads
type createEventRequest struct {
	ID        string  `json:"id"`
	EventName string  `json:"event_name"`
	Quota     int     `json:"quota"`
	Price     float64 `json:"price"`
}

type checkoutRequest struct {
	TicketID string `json:"ticket_id"`
	Quantity int    `json:"quantity"`
}

// HTTP Handlers
// CreateEvent godoc
// @Summary		Create a new event ticket
// @Description	Creates a new ticket inventory item in PostgreSQL
// @Tags		Tickets
// @Accept		json
// @Produce		json
// @Param		request body createEventRequest true "Event Ticket Creation Payload"
// @Success		201 {object} JSONResponse
// @Failure		400 {object} ErrorResponse
// @Failure		500 {object} ErrorResponse
// @Router		/tickets [post]
func (h *TicketHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req createEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	ticket := &domain.Ticket{
		EventName: req.EventName,
		Quota:     req.Quota,
		Price:     req.Price,
	}

	err := h.ticketUC.CreateEvent(r.Context(), ticket)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusCreated, JSONResponse{
		Status:  "success",
		Message: "Event created successfully",
	})
}

// BrowseAvailable godoc
// @Summary		Browse available tickets
// @Description	Fetches active tickets with Redis Cache-Aside pattern
// @Tags		Tickets
// @Produce		json
// @Success		200 {object} JSONResponse
// @Failure		500 {object} ErrorResponse
// @Router		/events [get]
func (h *TicketHandler) BrowseAvailable(w http.ResponseWriter, r *http.Request) {
	tickets, err := h.ticketUC.BrowseAvailable(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, JSONResponse{
		Status: "success",
		Data:   tickets,
	})
}

// CheckoutTicket godoc
// @Summary		Direct Ticket Checkout
// @Description	Decreases ticket quota directly with anti race-condition OCC handling
// @Tags		Tickets
// @Accept		json
// @Produce		json
// @Param		request body checkoutRequest true "Checkout Payload"
// @Success		200 {object} JSONResponse
// @Failure		400 {object} ErrorResponse
// @Failure		404 {object} ErrorResponse
// @Failure		409 {object} ErrorResponse
// @Failure		500 {object} ErrorResponse
// @Router		/tickets/checkout [post]
func (h *TicketHandler) CheckoutTicket(w http.ResponseWriter, r *http.Request) {
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	err := h.ticketUC.CheckoutTicket(r.Context(), req.TicketID, req.Quantity)
	if err != nil {
		switch err {
		case domain.ErrTicketUnavailable:
			writeError(w, http.StatusConflict, err) // HTTP 409 => Conflict for Race Condition / Stock Sold Out
		case domain.ErrTicketNotFound:
			writeError(w, http.StatusNotFound, err) // HTTP 404
		default:
			writeError(w, http.StatusInternalServerError, err) // HTTP 500
		}
		return
	}

	writeJSON(w, http.StatusOK, JSONResponse{
		Status:  "success",
		Message: "Ticket successfully checked out",
	})
}
