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
	ID			string	`json:"id"`
	EventName	string	`json:"event_name"`
	Quota		int		`json:"quota"`
	Price		float64	`json:"price"`
}

type checkoutRequest struct {
	TicketID	string	`json:"ticket_id"`
	Quantity	int		`json:"quantity"`
}

type jsonResponse struct {
	Status	string		`json:"status"`
	Message	string		`json:"message,omitempty"`
	Data	interface{}	`json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, jsonResponse{
		Status:  "error",
		Message: err.Error(),
	})
}

// HTTP Handlers
func (h *TicketHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req createEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	ticket := &domain.Ticket{
		ID:			req.ID,
		EventName:	req.EventName,
		Quota:		req.Quota,
		Price:		req.Price,
	}

	err := h.ticketUC.CreateEvent(r.Context(), ticket)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusCreated, jsonResponse{
		Status:		"success",
		Message:	"Event created successfully",
	})
}

func (h *TicketHandler) BrowseAvailable(w http.ResponseWriter, r *http.Request) {
	tickets, err := h.ticketUC.BrowseAvailable(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, jsonResponse{
		Status:	"success",
		Data:	tickets,
	})
}

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

	writeJSON(w, http.StatusOK, jsonResponse{
		Status:		"success",
		Message:	"Ticket successfully checked out",
	})
}