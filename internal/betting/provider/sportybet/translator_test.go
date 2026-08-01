package sportybet

import (
	"context"
	"testing"
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

	payload := SportyBetPayload{
		Data: struct {
			Ticket   SportyBetTicket `json:"ticket"`
			Outcomes []SportyBetItem `json:"outcomes"`
		}{
			Ticket: SportyBetTicket{
				Selections: []SportyBetTicketSelection{
					{EventID: "sr:match:72868102", MarketID: "m1", OutcomeID: "o1"},
					{EventID: "sr:match:72868102", MarketID: "m2", OutcomeID: "o2"},
				},
			},
			Outcomes: input,
		},
	}
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
	jsonPayload := []byte(`{"data":{"ticket":{"selections":[]},"outcomes":[]}}`)
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
}

func TestFetchAndParse_InvalidJSON(t *testing.T) {
	client := &fakeCloudflareClient{response: []byte("invalid json")}
	provider := NewProvider(client)

	_, err := provider.FetchAndParse(context.Background(), "J6J2TN")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
