package service

import (
	"context"

	"github.com/google/uuid"

	"sportloga/internal/betting/domain"
	db "sportloga/internal/db/generated"
)

// TicketProvider is the interface every provider package must implement.
// Adding a new bookie means writing a new package that satisfies this —
// no changes to BettingService required.
type TicketProvider interface {
	FetchAndParse(ctx context.Context, shareCode string) (*domain.SportlogaTicket, error)
}

// Repository defines the interface for database operations.
type Repository interface {
	UpsertGlobalTicket(ctx context.Context, ticket *domain.SportlogaTicket) (uuid.UUID, error)
	UpsertUserTrack(ctx context.Context, userID, bookingCodeID uuid.UUID, stake *float64, description string) error
	GetUserHistory(ctx context.Context, userID uuid.UUID) ([]db.GetUserHistoryRow, error)
	GetTicketDetails(ctx context.Context, arg db.GetTicketDetailsParams) ([]db.GetTicketDetailsRow, error)
	DeleteUserTicket(ctx context.Context, arg db.DeleteUserTicketParams) error
	CleanupOrphanedBookingCodes(ctx context.Context) error
}

// BettingService is the main manager for the betting module.
type BettingService struct {
	repo      Repository
	providers map[string]TicketProvider
}

// NewBettingService creates a new centralized BettingService.
func NewBettingService(repo Repository, providers map[string]TicketProvider) *BettingService {
	return &BettingService{
		repo:      repo,
		providers: providers,
	}
}

// PreviewResult is the result returned by PreviewTicket.
type PreviewResult struct {
	BookingCodeID uuid.UUID
	Ticket        *domain.SportlogaTicket
}

// PreviewTicket fetches JSON from Cloudflare, translates it into a ticket struct,
// and upserts it globally into booking_codes, matches, and booking_selections.
func (s *BettingService) PreviewTicket(ctx context.Context, provider string, shareCode string) (*PreviewResult, error) {
	p, ok := s.providers[provider]
	if !ok {
		return nil, ErrUnsupportedProvider
	}

	ticket, err := p.FetchAndParse(ctx, shareCode)
	if err != nil {
		return nil, err
	}

	bookingCodeID, err := s.repo.UpsertGlobalTicket(ctx, ticket)
	if err != nil {
		return nil, err
	}

	return &PreviewResult{
		BookingCodeID: bookingCodeID,
		Ticket:        ticket,
	}, nil
}

// TrackTicket associates a previously previewed booking code with a user.
func (s *BettingService) TrackTicket(ctx context.Context, userID, bookingCodeID uuid.UUID, stake *float64, description string) error {
	return s.repo.UpsertUserTrack(ctx, userID, bookingCodeID, stake, description)
}

// GetHistory fetches the user's tracked tickets.
func (s *BettingService) GetHistory(ctx context.Context, userID uuid.UUID) ([]db.GetUserHistoryRow, error) {
	return s.repo.GetUserHistory(ctx, userID)
}

// GetTicketDetails fetches the details of a specific tracked ticket.
func (s *BettingService) GetTicketDetails(ctx context.Context, arg db.GetTicketDetailsParams) ([]db.GetTicketDetailsRow, error) {
	return s.repo.GetTicketDetails(ctx, arg)
}

// DeleteTicket removes the link between a user and a ticket.
func (s *BettingService) DeleteTicket(ctx context.Context, arg db.DeleteUserTicketParams) error {
	return s.repo.DeleteUserTicket(ctx, arg)
}

// CleanupOrphanedTickets cleans up orphaned booking codes.
func (s *BettingService) CleanupOrphanedTickets(ctx context.Context) error {
	return s.repo.CleanupOrphanedBookingCodes(ctx)
}
