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
	HomeTeam   string  `json:"home_team"`
	AwayTeam   string  `json:"away_team"`
	MarketType string  `json:"market_type"`
	Selection  string  `json:"selection"`
	Odds       float64 `json:"odds"`
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
}

// TicketDetailItem represents a single selection detail when viewing a tracked ticket.
type TicketDetailItem struct {
	SelectionID     string  `json:"selection_id"`
	MarketType      string  `json:"market_type"`
	MarketSpec      string  `json:"market_spec"`
	Selection       string  `json:"selection"`
	Odds            float64 `json:"odds"`
	SelectionStatus string  `json:"selection_status"`
	HomeTeam        string  `json:"home_team"`
	AwayTeam        string  `json:"away_team"`
	StartTime       string  `json:"start_time"`
	MatchStatus     string  `json:"match_status"`
}
