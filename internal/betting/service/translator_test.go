package service

import (
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
					Desc:      "Handicap 2:0",
					Specifier: "hcp=2:0",
					Outcomes: []SportyBetOutcome{
						{
							Desc: "Home (2:0)",
							Odds: "1.06",
						},
					},
				},
				{
					Desc:      "Over/Under",
					Specifier: "total=2.5",
					Outcomes: []SportyBetOutcome{
						{
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
					{EventID: "1", MarketID: "1", OutcomeID: "1"},
					{EventID: "2", MarketID: "2", OutcomeID: "2"},
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
