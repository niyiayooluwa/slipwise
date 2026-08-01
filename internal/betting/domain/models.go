package domain

type Match struct {
	ExternalMatchID string
	HomeTeam        string
	AwayTeam        string
	StartTime       int64
}

type BookingSelection struct {
	Provider        string
	ExternalMatchID string
	MarketType      string
	MarketSpec      string
	Selection       string
	Odds            float64
	Status          string
}
