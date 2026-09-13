// Package service implements betting logic.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"slipwise/internal/betting/domain"
	db "slipwise/internal/db/generated"
)

// TicketProvider is the interface every provider package must implement.
// Adding a new bookie means writing a new package that satisfies this —
// no changes to BettingService required.
type TicketProvider interface {
	FetchAndParse(ctx context.Context, shareCode string) (*domain.SlipwiseTicket, error)
}

// Repository defines the interface for database operations.
type Repository interface {
	UpsertGlobalTicket(ctx context.Context, ticket *domain.SlipwiseTicket) (uuid.UUID, error)
	UpsertUserTrack(ctx context.Context, userID, bookingCodeID uuid.UUID, stake *float64, description string) error
	GetUserStats(ctx context.Context, userID uuid.UUID) (db.GetUserStatsRow, error)
	GetUserHistory(ctx context.Context, arg db.GetUserHistoryParams) ([]db.GetUserHistoryRow, error)
	CountUserHistory(ctx context.Context, arg db.CountUserHistoryParams) (int64, error)
	GetTicketDetails(ctx context.Context, arg db.GetTicketDetailsParams) ([]db.GetTicketDetailsRow, error)
	DeleteUserTicket(ctx context.Context, arg db.DeleteUserTicketParams) error
	BulkArchiveUserTickets(ctx context.Context, userID uuid.UUID, ticketIDs []uuid.UUID) (int64, error)
	BulkUnarchiveUserTickets(ctx context.Context, userID uuid.UUID, ticketIDs []uuid.UUID) (int64, error)
	BulkSoftDeleteUserTickets(ctx context.Context, userID uuid.UUID, ticketIDs []uuid.UUID) (int64, error)
	UpdateUserTicket(ctx context.Context, arg db.UpdateUserTicketParams) (db.UserTicket, error)
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
	Ticket        *domain.SlipwiseTicket
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

// GetUserStats fetches the user's materialized betting stats.
func (s *BettingService) GetUserStats(ctx context.Context, userID uuid.UUID) (db.GetUserStatsRow, error) {
	return s.repo.GetUserStats(ctx, userID)
}

// GetHistory fetches the user's tracked tickets with pagination, optional status filter, delta sync timestamp, and archive filter.
func (s *BettingService) GetHistory(ctx context.Context, userID uuid.UUID, limit, offset int32, status string, since *time.Time, isArchived bool) ([]db.GetUserHistoryRow, int64, error) {
	var statusArg *string
	if status != "" {
		statusArg = &status
	}

	rows, err := s.repo.GetUserHistory(ctx, db.GetUserHistoryParams{
		UserID:     userID,
		Limit:      limit,
		Offset:     offset,
		IsArchived: pgtype.Bool{Bool: isArchived, Valid: true},
		Status:     statusArg,
		Since:      since,
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.CountUserHistory(ctx, db.CountUserHistoryParams{
		UserID:     userID,
		IsArchived: pgtype.Bool{Bool: isArchived, Valid: true},
		Status:     statusArg,
	})
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

// parseUUIDs validates and converts raw string ticket IDs into UUIDs.
func parseUUIDs(rawIDs []string) ([]uuid.UUID, error) {
	if len(rawIDs) == 0 {
		return nil, errors.New("no ticket IDs provided")
	}
	res := make([]uuid.UUID, 0, len(rawIDs))
	for _, idStr := range rawIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ticket ID: %s", idStr)
		}
		res = append(res, id)
	}
	return res, nil
}

// BulkArchiveTickets archives the specified tickets for the user.
func (s *BettingService) BulkArchiveTickets(ctx context.Context, userID uuid.UUID, rawIDs []string) (int64, error) {
	ids, err := parseUUIDs(rawIDs)
	if err != nil {
		return 0, err
	}
	return s.repo.BulkArchiveUserTickets(ctx, userID, ids)
}

// BulkUnarchiveTickets unarchives/restores the specified tickets for the user.
func (s *BettingService) BulkUnarchiveTickets(ctx context.Context, userID uuid.UUID, rawIDs []string) (int64, error) {
	ids, err := parseUUIDs(rawIDs)
	if err != nil {
		return 0, err
	}
	return s.repo.BulkUnarchiveUserTickets(ctx, userID, ids)
}

// BulkDeleteTickets soft-deletes the specified tickets for the user without corrupting stats.
func (s *BettingService) BulkDeleteTickets(ctx context.Context, userID uuid.UUID, rawIDs []string) (int64, error) {
	ids, err := parseUUIDs(rawIDs)
	if err != nil {
		return 0, err
	}
	return s.repo.BulkSoftDeleteUserTickets(ctx, userID, ids)
}

// UpdateTicket updates a tracked ticket's description or stake.
func (s *BettingService) UpdateTicket(ctx context.Context, arg db.UpdateUserTicketParams) (db.UserTicket, error) {
	return s.repo.UpdateUserTicket(ctx, arg)
}

// GetTicketDetails fetches the details of a specific tracked ticket.
func (s *BettingService) GetTicketDetails(ctx context.Context, arg db.GetTicketDetailsParams) ([]db.GetTicketDetailsRow, error) {
	return s.repo.GetTicketDetails(ctx, arg)
}

// DeleteTicket soft-deletes a single ticket for the user.
func (s *BettingService) DeleteTicket(ctx context.Context, arg db.DeleteUserTicketParams) error {
	return s.repo.DeleteUserTicket(ctx, arg)
}

// CleanupOrphanedTickets cleans up orphaned booking codes.
func (s *BettingService) CleanupOrphanedTickets(ctx context.Context) error {
	return s.repo.CleanupOrphanedBookingCodes(ctx)
}
