package domain

import "errors"

var (
	ErrInvalidEventName	= errors.New("event name cannot be empty")
	ErrInvalidQuota		= errors.New("ticket quota must be greater than zero")
	ErrInvalidPrice		= errors.New("ticket price cannot be negative")

	ErrTicketNotFound		= errors.New("ticket not found")
	ErrTicketUnavailable	= errors.New("ticket is already sold or locked")
)