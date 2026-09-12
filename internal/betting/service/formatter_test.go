package service

import (
	"strings"
	"testing"
)

func TestFormatDisplaySelection(t *testing.T) {
	home := "Arsenal"
	away := "Chelsea"

	tests := []struct {
		name       string
		marketType string
		marketSpec string
		selection  string
		expected   string
	}{
		{
			name:       "1st Half Over",
			marketType: "1ST_HALF_OVER_UNDER",
			marketSpec: "total=1.5",
			selection:  "OVER",
			expected:   "1st Half Over 1.5 Goals",
		},
		{
			name:       "2nd Half Under",
			marketType: "2ND_HALF_OVER_UNDER",
			marketSpec: "0.5",
			selection:  "UNDER",
			expected:   "2nd Half Under 0.5 Goals",
		},
		{
			name:       "1st Half Match Result Home",
			marketType: "1ST_HALF_MATCH_RESULT",
			marketSpec: "",
			selection:  "1",
			expected:   "Arsenal to Win 1st Half",
		},
		{
			name:       "1st Half Match Result Draw",
			marketType: "1ST_HALF_MATCH_RESULT",
			marketSpec: "",
			selection:  "X",
			expected:   "Draw (1st Half)",
		},
		{
			name:       "1st Half Double Chance",
			marketType: "1ST_HALF_DOUBLE_CHANCE",
			marketSpec: "",
			selection:  "1X",
			expected:   "Arsenal or Draw (1st Half)",
		},
		{
			name:       "1st Half BTTS",
			marketType: "1ST_HALF_BTTS",
			marketSpec: "",
			selection:  "YES",
			expected:   "Both Teams to Score in 1st Half",
		},
		{
			name:       "Home Over Under",
			marketType: "HOME_OVER_UNDER",
			marketSpec: "total=1.5",
			selection:  "OVER",
			expected:   "Arsenal Over 1.5 Goals",
		},
		{
			name:       "Away Over Under",
			marketType: "AWAY_OVER_UNDER",
			marketSpec: "total=0.5",
			selection:  "OVER",
			expected:   "Chelsea Over 0.5 Goals",
		},
		{
			name:       "Home or Over",
			marketType: "HOME_OR_OVER",
			marketSpec: "total=2.5",
			selection:  "YES",
			expected:   "Arsenal to Win or Over 2.5 Goals",
		},
		{
			name:       "Away or Over",
			marketType: "AWAY_OR_OVER",
			marketSpec: "total=1.5",
			selection:  "YES",
			expected:   "Chelsea to Win or Over 1.5 Goals",
		},
		{
			name:       "Full-time Over Under",
			marketType: "OVER_UNDER",
			marketSpec: "total=2.5",
			selection:  "OVER",
			expected:   "Over 2.5 Goals",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDisplaySelection(tt.marketType, tt.marketSpec, tt.selection, home, away)
			if got != tt.expected {
				t.Errorf("FormatDisplaySelection() = %q, want %q", got, tt.expected)
			}
			if strings.Contains(got, "_") {
				t.Errorf("FormatDisplaySelection() leaked underscores: %q", got)
			}
		})
	}
}

func TestFormatDisplaySelection_FallbackNoUnderscores(t *testing.T) {
	got := FormatDisplaySelection("SOME_SPECIAL_FEATURE_MARKET", "", "YES", "Arsenal", "Chelsea")
	if strings.Contains(got, "_") {
		t.Errorf("Fallback leaked underscores: %q", got)
	}
	if !strings.Contains(got, "Some Special Feature Market") {
		t.Errorf("Expected title-cased fallback, got: %q", got)
	}
}
