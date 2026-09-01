package domain

import (
	"time"

	"github.com/google/uuid"
)

// SlipwiseTicket represents a full accumulator ticket saved by a user.
type SlipwiseTicket struct {
	UserID uuid.UUID
	// Stake is a pointer so it can be NULL.
	// Why? If a user is tracking a ticket just to monitor their friend's picks (social tracking),
	// they haven't risked money, so stake=nil. If stake is set, it computes into personal ROI/profit stats.
	Stake       *float64
	Description string
	Provider    string
	Code        string
	TotalOdds   float64
	Selections  []TicketSelection
}

// TicketSelection holds a single leg on a user's tracked ticket with full match metadata.
type TicketSelection struct {
	Match           MatchDetails
	ExternalMatchID string
	MarketType      string
	MarketSpec      *string
	Selection       string
	Odds            float64
}

// MatchDetails contains team names and kickoff time for client rendering.
type MatchDetails struct {
	HomeTeam  string
	AwayTeam  string
	StartTime time.Time
}
