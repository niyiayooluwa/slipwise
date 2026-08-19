package service

import (
	"testing"

	"slipwise/internal/betting/domain"
)

func TestEvaluateSelection(t *testing.T) {
	tests := []struct {
		name     string
		sel      domain.BookingSelection
		score    MatchScore
		expected string
	}{
		{
			name:     "Match Result Home Win",
			sel:      domain.BookingSelection{MarketType: "MATCH_RESULT", Selection: "1"},
			score:    MatchScore{HomeScoreFT: 2, AwayScoreFT: 1},
			expected: "WON",
		},
		{
			name:     "Match Result Away Win Lost",
			sel:      domain.BookingSelection{MarketType: "MATCH_RESULT", Selection: "2"},
			score:    MatchScore{HomeScoreFT: 2, AwayScoreFT: 1},
			expected: "LOST",
		},
		{
			name:     "Over Under Won",
			sel:      domain.BookingSelection{MarketType: "OVER_UNDER", MarketSpec: "2.5", Selection: "OVER"},
			score:    MatchScore{HomeScoreFT: 2, AwayScoreFT: 1}, // total 3
			expected: "WON",
		},
		{
			name:     "Asian Handicap Half Won",
			sel:      domain.BookingSelection{MarketType: "ASIAN_HANDICAP", MarketSpec: "-0.75", Selection: "1"},
			score:    MatchScore{HomeScoreFT: 1, AwayScoreFT: 0},
			expected: "HALF_WON", // diff = 1 - 0.75 - 0 = 0.25 -> HALF_WON
		},
		{
			name:     "Asian Handicap Void",
			sel:      domain.BookingSelection{MarketType: "ASIAN_HANDICAP", MarketSpec: "-1", Selection: "1"},
			score:    MatchScore{HomeScoreFT: 1, AwayScoreFT: 0},
			expected: "VOID", // diff = 1 - 1 - 0 = 0 -> VOID
		},
		{
			name:     "Asian Handicap Half Lost",
			sel:      domain.BookingSelection{MarketType: "ASIAN_HANDICAP", MarketSpec: "-1.25", Selection: "1"},
			score:    MatchScore{HomeScoreFT: 1, AwayScoreFT: 0},
			expected: "HALF_LOST", // diff = 1 - 1.25 - 0 = -0.25 -> HALF_LOST
		},
		{
			name:     "Handicap Won",
			sel:      domain.BookingSelection{MarketType: "HANDICAP", MarketSpec: "0:2", Selection: "1"},
			score:    MatchScore{HomeScoreFT: 3, AwayScoreFT: 0},
			expected: "WON",
		},
		{
			name:     "HT_FT Won",
			sel:      domain.BookingSelection{MarketType: "HT_FT", Selection: "1/X"},
			score:    MatchScore{HomeScoreHT: 1, AwayScoreHT: 0, HomeScoreFT: 1, AwayScoreFT: 1},
			expected: "WON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateSelection(tt.sel, tt.score)
			if got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}
