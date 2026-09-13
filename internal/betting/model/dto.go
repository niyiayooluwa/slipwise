// Package model holds the betting domain's request/response DTOs.
// These are kept separate from sqlc-generated structs so the DB
// schema can change without breaking the public API shape.
package model

// PreviewRequest is the body for POST /v1/tickets/preview.
type PreviewRequest struct {
	Provider string `json:"provider" example:"SPORTYBET"`
	Code     string `json:"code"     example:"J6J2TN"`
}

// SelectionDetail is a single game on the previewed ticket.
type SelectionDetail struct {
	HomeTeam         string  `json:"home_team"`
	AwayTeam         string  `json:"away_team"`
	MarketType       string  `json:"market_type"`
	MarketSpec       string  `json:"market_spec,omitempty"`
	Selection        string  `json:"selection"`
	DisplaySelection string  `json:"display_selection"`
	Odds             float64 `json:"odds"`
}

// PreviewResponse is returned by POST /v1/tickets/preview.
type PreviewResponse struct {
	BookingCodeID string            `json:"booking_code_id"`
	Provider      string            `json:"provider"`
	Code          string            `json:"code"`
	TotalOdds     float64           `json:"total_odds"`
	Selections    []SelectionDetail `json:"selections"`
}

// TrackRequest is the body for POST /v1/tickets/track.
type TrackRequest struct {
	BookingCodeID string   `json:"booking_code_id"`
	Stake         *float64 `json:"stake,omitempty"` // null = stakeless social tracking
	Description   string   `json:"description"`
}

// HistoryItem is a single ticket in the user's history.
type HistoryItem struct {
	TicketID      string   `json:"ticket_id"`
	Stake         *float64 `json:"stake"`
	Description   string   `json:"description"`
	TrackedAt     string   `json:"tracked_at"`
	Provider      string   `json:"provider"`
	Code          string   `json:"code"`
	TotalOdds     float64  `json:"total_odds"`
	OverallStatus string   `json:"overall_status"`
	TotalLegs     int32    `json:"total_legs"`
	WonLegs       int32    `json:"won_legs"`
	LostLegs      int32    `json:"lost_legs"`
	PendingLegs   int32    `json:"pending_legs"`
}

// TicketSummary contains aggregate counts of selections on a ticket.
type TicketSummary struct {
	TotalLegs   int32 `json:"total_legs"`
	WonLegs     int32 `json:"won_legs"`
	LostLegs    int32 `json:"lost_legs"`
	PendingLegs int32 `json:"pending_legs"`
}

// TicketDetailItem represents a single selection detail when viewing a tracked ticket.
type TicketDetailItem struct {
	SelectionID      string  `json:"selection_id"`
	MarketType       string  `json:"market_type"`
	MarketSpec       string  `json:"market_spec"`
	Selection        string  `json:"selection"`
	DisplaySelection string  `json:"display_selection"`
	Odds             float64 `json:"odds"`
	SelectionStatus  string  `json:"selection_status"`
	HomeTeam         string  `json:"home_team"`
	AwayTeam         string  `json:"away_team"`
	StartTime        string  `json:"start_time"`
	MatchStatus      string  `json:"match_status"`
	HomeScore        int32   `json:"home_score"`
	AwayScore        int32   `json:"away_score"`
	LiveTime         *string `json:"live_time"`
}

// TicketDetailsResponse is the response body for GET /v1/tickets/{id}.
type TicketDetailsResponse struct {
	Summary    TicketSummary      `json:"summary"`
	Selections []TicketDetailItem `json:"selections"`
}

// PaginationMeta holds metadata for paginated responses.
type PaginationMeta struct {
	Total   int64 `json:"total"`
	Page    int32 `json:"page"`
	Limit   int32 `json:"limit"`
	HasNext bool  `json:"has_next"`
}

// PaginatedHistoryResponse is the paginated response for ticket history.
type PaginatedHistoryResponse struct {
	Data []HistoryItem  `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// UserStatsResponse represents a user's materialized betting stats.
type UserStatsResponse struct {
	TotalTickets   int32   `json:"total_tickets"`
	WonTickets     int32   `json:"won_tickets"`
	LostTickets    int32   `json:"lost_tickets"`
	PendingTickets int32   `json:"pending_tickets"`
	TotalStaked    float64 `json:"total_staked"`
	TotalReturns   float64 `json:"total_returns"`
	NetProfit      float64 `json:"net_profit"`
}

// BulkTicketActionRequest is the payload for batch archiving, unarchiving, and soft-deleting tickets.
type BulkTicketActionRequest struct {
	TicketIDs []string `json:"ticket_ids" example:"[\"3fa85f64-5717-4562-b3fc-2c963f66afa6\"]"`
}

// BulkTicketActionResponse returns the number of affected tickets from a bulk operation.
type BulkTicketActionResponse struct {
	Affected int64  `json:"affected" example:"3"`
	Message  string `json:"message" example:"tickets archived successfully"`
}
