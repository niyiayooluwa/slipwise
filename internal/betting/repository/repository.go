// Package repository handles database operations for the betting system.
package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"slipwise/internal/betting/domain"
	db "slipwise/internal/db/generated"
	"slipwise/internal/worker"
)

type Repository struct {
	dbPool  *pgxpool.Pool
	queries *db.Queries
}

func NewRepository(dbPool *pgxpool.Pool) *Repository {
	return &Repository{
		dbPool:  dbPool,
		queries: db.New(dbPool),
	}
}

func (r *Repository) UpsertGlobalTicket(ctx context.Context, ticket *domain.SlipwiseTicket) (uuid.UUID, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := r.queries.WithTx(tx)

	// Create or update booking code
	bc, err := q.CreateBookingCode(ctx, db.CreateBookingCodeParams{
		Provider:  ticket.Provider,
		Code:      ticket.Code,
		TotalOdds: floatToNumeric(ticket.TotalOdds),
		Status:    "PENDING",
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to upsert booking code: %w", err)
	}

	// Create matches and selections
	for _, sel := range ticket.Selections {
		// Create or update match

		dbMatch, err := q.CreateMatch(ctx, db.CreateMatchParams{
			HomeTeam:   sel.Match.HomeTeam,
			AwayTeam:   sel.Match.AwayTeam,
			StartTime:  pgtype.Timestamptz{Time: sel.Match.StartTime, Valid: true},
			Status:     "PENDING",
			Provider:   ticket.Provider,
			ProviderID: sel.ExternalMatchID,
		})
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to create match: %w", err)
		}

		_, err = q.CreateBookingSelection(ctx, db.CreateBookingSelectionParams{
			BookingCodeID: bc.ID,
			MatchID:       dbMatch.ID,
			MarketType:    sel.MarketType,
			MarketSpec:    sel.MarketSpec,
			Selection:     sel.Selection,
			Odds:          floatToNumeric(sel.Odds),
			Status:        "PENDING",
		})
		if err != nil {
			// Because of the ON CONFLICT missing in selection? Actually if they preview same code twice, it might fail on duplicate selection if there's a constraint, but assuming we can just ignore or let it pass for now.
			return uuid.Nil, fmt.Errorf("failed to create booking selection: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return bc.ID, nil
}

func (r *Repository) UpsertUserTrack(ctx context.Context, userID, bookingCodeID uuid.UUID, stake *float64, description string) error {
	_, err := r.queries.UpsertUserTrack(ctx, db.UpsertUserTrackParams{
		UserID:        userID,
		BookingCodeID: bookingCodeID,
		Stake:         floatPtrToNumeric(stake),
		Description:   description,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert user track: %w", err)
	}
	return nil
}

func (r *Repository) UpdateSelectionStatus(ctx context.Context, arg db.UpdateSelectionStatusParams) ([]uuid.UUID, error) {
	return r.queries.UpdateSelectionStatus(ctx, arg)
}

func (r *Repository) DeleteUserTicket(ctx context.Context, arg db.DeleteUserTicketParams) error {
	return r.queries.DeleteUserTicket(ctx, arg)
}

func (r *Repository) GetUserHistory(ctx context.Context, arg db.GetUserHistoryParams) ([]db.GetUserHistoryRow, error) {
	return r.queries.GetUserHistory(ctx, arg)
}

func (r *Repository) CountUserHistory(ctx context.Context, userID uuid.UUID) (int64, error) {
	return r.queries.CountUserHistory(ctx, userID)
}

func (r *Repository) GetTicketDetails(ctx context.Context, arg db.GetTicketDetailsParams) ([]db.GetTicketDetailsRow, error) {
	return r.queries.GetTicketDetails(ctx, arg)
}

func (r *Repository) GetPendingBucketsForMatch(ctx context.Context, matchID uuid.UUID) ([]db.GetPendingBucketsForMatchRow, error) {
	return r.queries.GetPendingBucketsForMatch(ctx, matchID)
}

func (r *Repository) CleanupOrphanedBookingCodes(ctx context.Context) error {
	return r.queries.CleanupOrphanedBookingCodes(ctx)
}

func (r *Repository) GetPendingMatches(ctx context.Context) ([]worker.PendingMatch, error) {
	// Hardcoded to SPORTYBET for now since it's the main provider for polling
	rows, err := r.queries.GetActiveBucketsByProvider(ctx, "SPORTYBET")
	if err != nil {
		return nil, err
	}
	var matches []worker.PendingMatch
	for _, row := range rows {
		matches = append(matches, worker.PendingMatch{
			MatchID:    row.ID,
			ProviderID: row.ProviderID,
		})
	}
	return matches, nil
}

// Helpers
func floatToNumeric(f float64) pgtype.Numeric {
	n := pgtype.Numeric{}
	n.Scan(fmt.Sprintf("%f", f))
	return n
}

func floatPtrToNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{Valid: false}
	}
	n := pgtype.Numeric{}
	n.Scan(fmt.Sprintf("%f", *f))
	return n
}
