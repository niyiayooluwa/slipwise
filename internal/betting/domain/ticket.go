package domain

import (
	"time"

	"github.com/google/uuid"
)

type SportlogaTicket struct {
	UserID     uuid.UUID
	Stake      float64
	Provider   string
	Code       string
	TotalOdds  float64
	Selections []TicketSelection
}

type TicketSelection struct {
	Match           MatchDetails
	ExternalMatchID string
	MarketType      string
	MarketSpec      *string
	Selection       string
	Odds            float64
}

type MatchDetails struct {
	HomeTeam  string
	AwayTeam  string
	StartTime time.Time
}
