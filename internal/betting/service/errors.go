package service

import "errors"

var (
	ErrUnsupportedProvider   = errors.New("unsupported provider")
	ErrTicketNotFound        = errors.New("booking code is invalid or not found")
	ErrTicketExpired         = errors.New("booking code has expired")
	ErrTicketAllMatchesEnded = errors.New("all matches on this ticket have already ended")
	ErrAlreadyTracking       = errors.New("you are already tracking this ticket")
)
