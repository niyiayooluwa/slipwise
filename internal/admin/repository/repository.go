// Package repository implements the data-access layer for the admin domain.
// It wraps sqlc-generated queries and returns plain Go types,
// keeping the rest of the admin domain completely decoupled from the DB layer.
package repository

import (
	"context"
	"time"

	generated "slipwise/internal/db/generated"

	"github.com/jackc/pgx/v5/pgtype"
)

// AdminUser is the repository-layer struct for a user record.
// It is NOT the HTTP DTO — the service maps it to model.AdminUserItem.
type AdminUser struct {
	ID              string
	Email           string
	Username        *string
	EmailVerifiedAt *time.Time
	IsAdmin         bool
	IsPunter        bool
	IsSuspended     bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// DashboardStats holds raw aggregated counts from the DB.
type DashboardStats struct {
	TotalUsers          int64
	TotalBookingCodes   int64
	TotalTrackedTickets int64
	WonTickets          int64
	LostTickets         int64
	PendingTickets      int64
}

// Repo is the interface the service layer depends on.
// Defined here so it can be mocked in service tests.
type Repo interface {
	GetDashboardStats(ctx context.Context) (DashboardStats, error)
	GetUsers(ctx context.Context, limit, offset int32) ([]AdminUser, error)
	GetUsersCount(ctx context.Context) (int64, error)
}

type repository struct {
	q *generated.Queries
}

// New returns a concrete Repo backed by sqlc-generated queries.
func New(q *generated.Queries) Repo {
	return &repository{q: q}
}

func (r *repository) GetDashboardStats(ctx context.Context) (DashboardStats, error) {
	row, err := r.q.GetAdminDashboardStats(ctx)
	if err != nil {
		return DashboardStats{}, err
	}
	return DashboardStats{
		TotalUsers:          row.TotalUsers,
		TotalBookingCodes:   row.TotalBookingCodes,
		TotalTrackedTickets: row.TotalTrackedTickets,
		WonTickets:          row.WonTickets,
		LostTickets:         row.LostTickets,
		PendingTickets:      row.PendingTickets,
	}, nil
}

func (r *repository) GetUsers(ctx context.Context, limit, offset int32) ([]AdminUser, error) {
	rows, err := r.q.GetAdminUsers(ctx, generated.GetAdminUsersParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}

	users := make([]AdminUser, 0, len(rows))
	for _, u := range rows {
		var username *string
		if u.Username != (pgtype.Text{}) && u.Username.Valid {
			username = &u.Username.String
		}
		users = append(users, AdminUser{
			ID:              u.ID.String(),
			Email:           u.Email,
			Username:        username,
			EmailVerifiedAt: u.EmailVerifiedAt,
			IsAdmin:         u.IsAdmin,
			IsPunter:        u.IsPunter,
			IsSuspended:     u.IsSuspended,
			CreatedAt:       u.CreatedAt,
			UpdatedAt:       u.UpdatedAt,
		})
	}
	return users, nil
}

func (r *repository) GetUsersCount(ctx context.Context) (int64, error) {
	return r.q.GetAdminUsersCount(ctx)
}
