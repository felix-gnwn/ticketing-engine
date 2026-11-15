package domain

import "errors"

var (
	ErrInvalidEventName = errors.New("event name cannot be empty")
	ErrInvalidQuota     = errors.New("ticket quota must be greater than zero")
	ErrInvalidPrice     = errors.New("ticket price cannot be negative")

	ErrTicketNotFound    = errors.New("ticket not found")
	ErrTicketUnavailable = errors.New("ticket is already sold or locked")

	ErrOrderNotFound     = errors.New("order not found")
	ErrInsufficientQuota = errors.New("insufficient ticket quota")
	ErrInvalidQuantity   = errors.New("order quantity must be greater than zero")
	ErrDuplicateOrder    = errors.New("duplicate order request detected (idempotent conflict)")
	ErrOrderAlreadyPaid  = errors.New("order is already paid")
	ErrOrderExpired      = errors.New("order has expired")
)
