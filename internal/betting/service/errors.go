package service

import "errors"

var (
	ErrUnsupportedProvider = errors.New("unsupported provider")
	ErrTicketNotFound      = errors.New("ticket not found")
	ErrAlreadyTracking     = errors.New("you are already tracking this ticket")
)
