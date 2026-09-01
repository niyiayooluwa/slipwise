// Package domain defines the core business entities and ubiquitous language
// for the SlipWise betting platform.
//
// Design Principle (Universal Market Representation):
// Different bookmakers (SportyBet, Bet9ja, 1xBet) use proprietary IDs, numbers, and weirdly
// nested structures for games and markets. The domain layer normalizes all of them into
// universal, clean concepts: Match, BookingSelection, and Ticket.
package domain

// Match represents a real-world sporting event across any bookmaker.
type Match struct {
	// ExternalMatchID is the bookmaker's identifier (e.g. "sr:match:51829103").
	ExternalMatchID string
	HomeTeam        string
	AwayTeam        string
	// StartTime is Unix timestamp in milliseconds when the match kicks off.
	StartTime int64
}

// BookingSelection represents a single betting pick/leg on a ticket.
type BookingSelection struct {
	// Provider identifies the source bookmaker (e.g. "SPORTYBET", "BET9JA").
	Provider        string
	ExternalMatchID string
	// MarketType is our universal enum string (e.g. "MATCH_RESULT", "OVER_UNDER", "BTTS").
	MarketType string
	// MarketSpec stores additional market modifiers (e.g. "2.5" for Over/Under, "+1" for Handicap).
	MarketSpec string
	// Selection is the normalized pick (e.g. "1", "X", "2", "OVER", "UNDER", "YES", "NO").
	Selection string
	Odds      float64
	// Status tracks settlement ("PENDING", "WON", "LOST", "VOID").
	Status string
}
