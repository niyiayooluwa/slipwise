// Package model holds the admin domain's HTTP request/response DTOs.
package model

import "time"

// DashboardStatsResponse represents the metrics shown on the admin dashboard.
type DashboardStatsResponse struct {
	TotalUsers          int64 `json:"total_users"`
	TotalBookingCodes   int64 `json:"total_booking_codes"`
	TotalTrackedTickets int64 `json:"total_tracked_tickets"`
	WonTickets          int64 `json:"won_tickets"`
	LostTickets         int64 `json:"lost_tickets"`
	PendingTickets      int64 `json:"pending_tickets"`
}

// AdminUserItem represents a single user in the admin user list.
type AdminUserItem struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Username    *string   `json:"username"`
	IsVerified  bool      `json:"is_verified"`
	IsAdmin     bool      `json:"is_admin"`
	IsPunter    bool      `json:"is_punter"`
	IsSuspended bool      `json:"is_suspended"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PaginationMeta holds metadata for paginated responses.
type PaginationMeta struct {
	Total   int64 `json:"total"`
	Page    int32 `json:"page"`
	Limit   int32 `json:"limit"`
	HasNext bool  `json:"has_next"`
}

// PaginatedAdminUsersResponse is the paginated response for the admin user list.
type PaginatedAdminUsersResponse struct {
	Data []AdminUserItem `json:"data"`
	Meta PaginationMeta  `json:"meta"`
}
