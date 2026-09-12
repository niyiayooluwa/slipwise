package sportybet

import (
	"context"
	"strconv"
	"testing"
	"time"

	"slipwise/internal/betting/service"
)

func TestTranslateSportyBet(t *testing.T) {
	input := []SportyBetItem{
		{
			EventID:           "sr:match:72868102",
			HomeTeamName:      "Kuopion Palloseura",
			AwayTeamName:      "Sabah Masazir",
			EstimateStartTime: 1785250800000,
			Markets: []SportyBetMarket{
				{
					ID:        "m1",
					Desc:      "Handicap 2:0",
					Specifier: "hcp=2:0",
					Outcomes: []SportyBetOutcome{
						{
							ID:   "o1",
							Desc: "Home (2:0)",
							Odds: "1.06",
						},
					},
				},
				{
					ID:        "m2",
					Desc:      "Over/Under",
					Specifier: "total=2.5",
					Outcomes: []SportyBetOutcome{
						{
							ID:   "o2",
							Desc: "Over",
							Odds: "1.80",
						},
					},
				},
			},
		},
	}

	var payload SportyBetPayload
	payload.BizCode = 10000
	payload.Data.ShareCode = "J6J2TN"
	payload.Data.Ticket.Selections = []SportyBetTicketSelection{
		{EventID: "sr:match:72868102", MarketID: "m1", OutcomeID: "o1"},
		{EventID: "sr:match:72868102", MarketID: "m2", OutcomeID: "o2"},
	}
	payload.Data.Outcomes = input
	matches, selections := TranslateSportyBet(payload)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].ExternalMatchID != "sr:match:72868102" {
		t.Errorf("unexpected match ID: %s", matches[0].ExternalMatchID)
	}

	if len(selections) != 2 {
		t.Fatalf("expected 2 selections, got %d", len(selections))
	}

	sel1 := selections[0]
	if sel1.MarketType != "HANDICAP" {
		t.Errorf("unexpected market type: %s", sel1.MarketType)
	}
	if sel1.MarketSpec != "2:0" {
		t.Errorf("unexpected specifier: %s", sel1.MarketSpec)
	}
	if sel1.Selection != "1" {
		t.Errorf("unexpected pick: %s", sel1.Selection)
	}
	if sel1.Odds != 1.06 {
		t.Errorf("unexpected odds: %f", sel1.Odds)
	}

	sel2 := selections[1]
	if sel2.MarketType != "OVER_UNDER" {
		t.Errorf("unexpected market type: %s", sel2.MarketType)
	}
	if sel2.MarketSpec != "2.5" {
		t.Errorf("unexpected specifier: %s", sel2.MarketSpec)
	}
	if sel2.Selection != "OVER" {
		t.Errorf("unexpected pick: %s", sel2.Selection)
	}
}

type fakeCloudflareClient struct {
	response []byte
	err      error
}

func (f *fakeCloudflareClient) FetchTicketByCode(ctx context.Context, shareCode string) ([]byte, error) {
	return f.response, f.err
}

func TestFetchAndParse_Success(t *testing.T) {
	futureTime := time.Now().Add(2 * time.Hour).UnixMilli()
	jsonPayload := []byte(`{
		"bizCode": 10000,
		"message": "Success",
		"data": {
			"shareCode": "J6J2TN",
			"ticket": {
				"selections": [{"eventId": "sr:match:123", "marketId": "m1", "outcomeId": "o1"}]
			},
			"outcomes": [{
				"eventId": "sr:match:123",
				"homeTeamName": "Arsenal",
				"awayTeamName": "Chelsea",
				"estimateStartTime": ` + strconv.FormatInt(futureTime, 10) + `,
				"matchStatus": "Not start",
				"status": 0,
				"markets": [{
					"id": "m1",
					"desc": "1X2",
					"outcomes": [{"id": "o1", "desc": "Home", "odds": "1.85"}]
				}]
			}]
		}
	}`)
	client := &fakeCloudflareClient{response: jsonPayload}
	provider := NewProvider(client)

	ticket, err := provider.FetchAndParse(context.Background(), "J6J2TN")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ticket.Code != "J6J2TN" {
		t.Errorf("expected code J6J2TN, got %s", ticket.Code)
	}
	if ticket.Provider != "SPORTYBET" {
		t.Errorf("expected provider SPORTYBET, got %s", ticket.Provider)
	}
	if len(ticket.Selections) != 1 {
		t.Errorf("expected 1 selection, got %d", len(ticket.Selections))
	}
}

func TestFetchAndParse_InvalidBizCode(t *testing.T) {
	jsonPayload := []byte(`{"bizCode": 19000, "message": "The code is invalid."}`)
	client := &fakeCloudflareClient{response: jsonPayload}
	provider := NewProvider(client)

	_, err := provider.FetchAndParse(context.Background(), "INVALID")
	if err != service.ErrTicketNotFound {
		t.Fatalf("expected ErrTicketNotFound, got %v", err)
	}
}

func TestFetchAndParse_AllMatchesEnded(t *testing.T) {
	jsonPayload := []byte(`{
		"bizCode": 10000,
		"message": "Success",
		"data": {
			"shareCode": "GRBA39",
			"ticket": {
				"selections": [{"eventId": "sr:match:123", "marketId": "m1", "outcomeId": "o1"}]
			},
			"outcomes": [{
				"eventId": "sr:match:123",
				"homeTeamName": "Team A",
				"awayTeamName": "Team B",
				"estimateStartTime": 1788595200000,
				"matchStatus": "Ended",
				"status": 3,
				"markets": [{
					"id": "m1",
					"desc": "1X2",
					"outcomes": [{"id": "o1", "desc": "Home", "odds": "1.05"}]
				}]
			}]
		}
	}`)
	client := &fakeCloudflareClient{response: jsonPayload}
	provider := NewProvider(client)

	_, err := provider.FetchAndParse(context.Background(), "GRBA39")
	if err != service.ErrTicketAllMatchesEnded {
		t.Fatalf("expected ErrTicketAllMatchesEnded, got %v", err)
	}
}

func TestFetchAndParse_InvalidJSON(t *testing.T) {
	client := &fakeCloudflareClient{response: []byte("invalid json")}
	provider := NewProvider(client)

	_, err := provider.FetchAndParse(context.Background(), "J6J2TN")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMapMarketType_HalfVariations(t *testing.T) {
	home := "Arsenal"
	away := "Chelsea"

	tests := []struct {
		desc     string
		expected string
	}{
		{"1st Half - Over/Under", "1ST_HALF_OVER_UNDER"},
		{"First Half - Over/Under", "1ST_HALF_OVER_UNDER"},
		{"1st Half - Total", "1ST_HALF_OVER_UNDER"},
		{"1st Half - 1X2", "1ST_HALF_MATCH_RESULT"},
		{"2nd Half - Over/Under", "2ND_HALF_OVER_UNDER"},
		{"Second Half - Over/Under", "2ND_HALF_OVER_UNDER"},
		{"2nd Half - 1X2", "2ND_HALF_MATCH_RESULT"},
		{"1st Half - GG/NG", "1ST_HALF_BTTS"},
		{"Chelsea or Over", "AWAY_OR_OVER"},
		{"Arsenal or Over", "HOME_OR_OVER"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := MapMarketType(tt.desc, home, away)
			if got != tt.expected {
				t.Errorf("MapMarketType(%q) = %q, want %q", tt.desc, got, tt.expected)
			}
		})
	}
}
