package repository

import (
	"context"
	"fmt"
	"math/big"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"sportloga/internal/betting/domain"
	db "sportloga/internal/db/generated"
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

func (r *Repository) SaveFullTicket(ctx context.Context, ticket domain.SportlogaTicket) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := r.queries.WithTx(tx)

	// Create booking code
	bc, err := q.CreateBookingCode(ctx, db.CreateBookingCodeParams{
		Provider:  ticket.Provider,
		Code:      ticket.Code,
		TotalOdds: floatToNumeric(ticket.TotalOdds),
		Status:    "PENDING",
	})
	if err != nil {
		return fmt.Errorf("failed to create booking code: %w", err)
	}

	// Create user ticket
	_, err = q.CreateUserTicket(ctx, db.CreateUserTicketParams{
		UserID:        ticket.UserID,
		BookingCodeID: bc.ID,
		Stake:         floatToNumeric(ticket.Stake),
	})
	if err != nil {
		return fmt.Errorf("failed to create user ticket: %w", err)
	}

	// Create matches and selections
	for _, sel := range ticket.Selections {
		// Attempt to create match. 
		// Note: In a real system, we'd check if match already exists by some external ID.
		// For the MVP plan, we just create it as requested.
		match, err := q.CreateMatch(ctx, db.CreateMatchParams{
			HomeTeam:  sel.Match.HomeTeam,
			AwayTeam:  sel.Match.AwayTeam,
			StartTime: pgtype.Timestamptz{Time: sel.Match.StartTime, Valid: true},
			Status:    "PENDING",
		})
		if err != nil {
			return fmt.Errorf("failed to create match: %w", err)
		}

		_, err = q.CreateBookingSelection(ctx, db.CreateBookingSelectionParams{
			BookingCodeID:   bc.ID,
			MatchID:         match.ID,
			Provider:        ticket.Provider,
			ExternalMatchID: sel.ExternalMatchID,
			MarketType:      sel.MarketType,
			MarketSpec:      sel.MarketSpec,
			Selection:       sel.Selection,
			Odds:            floatToNumeric(sel.Odds),
			Status:          "PENDING",
		})
		if err != nil {
			return fmt.Errorf("failed to create booking selection: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Queries exposes the raw queries for use outside of this package (e.g. UpdateSelectionStatus)
func (r *Repository) Queries() *db.Queries {
	return r.queries
}

// Helper to convert float64 to pgtype.Numeric
func floatToNumeric(f float64) pgtype.Numeric {
	s := fmt.Sprintf("%f", f)
	n := new(big.Int)
	n.SetString(s, 10) // Not perfect for decimal, but safe for float approximation if needed.
	// Actually better to parse the string with pgtype:
	num := pgtype.Numeric{}
	num.Scan(s)
	return num
}
