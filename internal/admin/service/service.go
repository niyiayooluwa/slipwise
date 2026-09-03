// Package service implements the business logic layer for the admin domain.
// It depends only on the repository interface — never on sqlc or pgx types.
package service

import (
	"context"

	adminmodel "slipwise/internal/admin/model"
	adminrepo "slipwise/internal/admin/repository"
)

// Service defines the contract the handler depends on.
// This interface is what you mock in handler tests.
type Service interface {
	GetDashboardStats(ctx context.Context) (*adminmodel.DashboardStatsResponse, error)
	GetUsers(ctx context.Context, search string, page, limit int) (*adminmodel.PaginatedAdminUsersResponse, error)
}

type service struct {
	repo adminrepo.Repo
}

// New returns a concrete Service backed by the given repository.
func New(repo adminrepo.Repo) Service {
	return &service{repo: repo}
}

func (s *service) GetDashboardStats(ctx context.Context) (*adminmodel.DashboardStatsResponse, error) {
	stats, err := s.repo.GetDashboardStats(ctx)
	if err != nil {
		return nil, err
	}
	return &adminmodel.DashboardStatsResponse{
		TotalUsers:          stats.TotalUsers,
		TotalBookingCodes:   stats.TotalBookingCodes,
		TotalTrackedTickets: stats.TotalTrackedTickets,
		WonTickets:          stats.WonTickets,
		LostTickets:         stats.LostTickets,
		PendingTickets:      stats.PendingTickets,
	}, nil
}

func (s *service) GetUsers(ctx context.Context, search string, page, limit int) (*adminmodel.PaginatedAdminUsersResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := int32((page - 1) * limit)

	users, err := s.repo.GetUsers(ctx, search, int32(limit), offset)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.GetUsersCount(ctx, search)
	if err != nil {
		return nil, err
	}

	items := make([]adminmodel.AdminUserItem, 0, len(users))
	for _, u := range users {
		items = append(items, adminmodel.AdminUserItem{
			ID:          u.ID,
			Email:       u.Email,
			Username:    u.Username,
			IsVerified:  u.EmailVerifiedAt != nil,
			IsAdmin:     u.IsAdmin,
			IsPunter:    u.IsPunter,
			IsSuspended: u.IsSuspended,
			CreatedAt:   u.CreatedAt,
			UpdatedAt:   u.UpdatedAt,
		})
	}

	return &adminmodel.PaginatedAdminUsersResponse{
		Data: items,
		Meta: adminmodel.PaginationMeta{
			Total:   total,
			Page:    int32(page),
			Limit:   int32(limit),
			HasNext: int64(page*limit) < total,
		},
	}, nil
}
