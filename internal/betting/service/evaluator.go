// Package service implements the core business logic for betting operations.
package service

import (
	"fmt"
	"strconv"
	"strings"

	"sportloga/internal/betting/domain"
)

type MatchScore struct {
	HomeScoreHT int
	AwayScoreHT int
	HomeScoreFT int
	AwayScoreFT int
}

// EvaluateSelection evaluates a single booking selection based on the match scores.
func EvaluateSelection(sel domain.BookingSelection, score MatchScore) string {
	home := score.HomeScoreFT
	away := score.AwayScoreFT
	homeHT := score.HomeScoreHT
	awayHT := score.AwayScoreHT
	total := home + away

	switch sel.MarketType {
	case "MATCH_RESULT":
		if home > away && sel.Selection == "1" {
			return "WON"
		}
		if home == away && sel.Selection == "X" {
			return "WON"
		}
		if home < away && sel.Selection == "2" {
			return "WON"
		}
		return "LOST"

	case "DOUBLE_CHANCE":
		if (home > away || home == away) && sel.Selection == "1X" {
			return "WON"
		}
		if (home > away || home < away) && sel.Selection == "12" {
			return "WON"
		}
		if (home < away || home == away) && sel.Selection == "X2" {
			return "WON"
		}
		return "LOST"

	case "DRAW_NO_BET":
		if home == away {
			return "VOID"
		}
		if home > away && sel.Selection == "1" {
			return "WON"
		}
		if home < away && sel.Selection == "2" {
			return "WON"
		}
		return "LOST"

	case "OVER_UNDER":
		spec, _ := strconv.ParseFloat(sel.MarketSpec, 64)
		if float64(total) > spec && sel.Selection == "OVER" {
			return "WON"
		}
		if float64(total) < spec && sel.Selection == "UNDER" {
			return "WON"
		}
		return "LOST"

	case "BTTS":
		btts := (home > 0 && away > 0)
		if btts && sel.Selection == "YES" {
			return "WON"
		}
		if !btts && sel.Selection == "NO" {
			return "WON"
		}
		return "LOST"

	case "CORRECT_SCORE":
		actual := fmt.Sprintf("%d:%d", home, away)
		if sel.Selection == actual || sel.MarketSpec == actual || sel.Selection == fmt.Sprintf("Home (%s)", actual) || sel.Selection == fmt.Sprintf("Away (%s)", actual) || sel.Selection == fmt.Sprintf("Draw (%s)", actual) {
			return "WON"
		}
		// Sportybet has correct score like 2:1 or 'Home (2:1)'
		// A bit fuzzy but if the string contains the actual score we could match it
		if strings.Contains(sel.Selection, actual) || strings.Contains(sel.MarketSpec, actual) {
			return "WON"
		}
		return "LOST"

	case "HANDICAP":
		hcpHome, hcpAway := parseHandicap(sel.MarketSpec)
		adjHome := float64(home) + hcpHome
		adjAway := float64(away) + hcpAway

		if adjHome > adjAway && sel.Selection == "1" {
			return "WON"
		}
		if adjHome == adjAway && sel.Selection == "X" {
			return "WON"
		}
		if adjHome < adjAway && sel.Selection == "2" {
			return "WON"
		}
		return "LOST"

	case "ASIAN_HANDICAP":
		hcp, _ := strconv.ParseFloat(sel.MarketSpec, 64)
		var diff float64
		if sel.Selection == "1" {
			diff = float64(home) + hcp - float64(away)
		} else {
			diff = float64(away) + hcp - float64(home)
		}

		if diff >= 0.5 {
			return "WON"
		}
		if diff == 0.25 {
			return "HALF_WON"
		}
		if diff == 0 {
			return "VOID"
		}
		if diff == -0.25 {
			return "HALF_LOST"
		}
		if diff <= -0.5 {
			return "LOST"
		}
		return "LOST"

	case "ASIAN_OVER_UNDER":
		spec, _ := strconv.ParseFloat(sel.MarketSpec, 64)
		diff := float64(total) - spec
		if sel.Selection == "OVER" {
			if diff >= 0.5 {
				return "WON"
			}
			if diff == 0.25 {
				return "HALF_WON"
			}
			if diff == 0 {
				return "VOID"
			}
			if diff == -0.25 {
				return "HALF_LOST"
			}
			if diff <= -0.5 {
				return "LOST"
			}
			return "LOST"
		} else if sel.Selection == "UNDER" {
			if diff <= -0.5 {
				return "WON"
			}
			if diff == -0.25 {
				return "HALF_WON"
			}
			if diff == 0 {
				return "VOID"
			}
			if diff == 0.25 {
				return "HALF_LOST"
			}
			if diff >= 0.5 {
				return "LOST"
			}
			return "LOST"
		}

	case "HOME_NO_BET":
		if home > away {
			return "VOID"
		}
		if home == away && sel.Selection == "X" {
			return "WON"
		}
		if home < away && sel.Selection == "2" {
			return "WON"
		}
		return "LOST"

	case "AWAY_NO_BET":
		if home < away {
			return "VOID"
		}
		if home > away && sel.Selection == "1" {
			return "WON"
		}
		if home == away && sel.Selection == "X" {
			return "WON"
		}
		return "LOST"

	case "HOME_OVER_UNDER":
		spec, _ := strconv.ParseFloat(sel.MarketSpec, 64)
		if float64(home) > spec && sel.Selection == "OVER" {
			return "WON"
		}
		if float64(home) < spec && sel.Selection == "UNDER" {
			return "WON"
		}
		return "LOST"

	case "AWAY_OVER_UNDER":
		spec, _ := strconv.ParseFloat(sel.MarketSpec, 64)
		if float64(away) > spec && sel.Selection == "OVER" {
			return "WON"
		}
		if float64(away) < spec && sel.Selection == "UNDER" {
			return "WON"
		}
		return "LOST"

	case "EXACT_GOALS":
		expected := strconv.Itoa(total)
		if total >= 5 {
			expected = "5+"
		}
		if sel.Selection == expected || sel.MarketSpec == expected {
			return "WON"
		}
		// support exact number matching
		if sel.Selection == strconv.Itoa(total) || sel.MarketSpec == strconv.Itoa(total) {
			return "WON"
		}
		return "LOST"

	case "GG2+":
		if home >= 2 && away >= 2 && sel.Selection == "YES" {
			return "WON"
		}
		return "LOST"

	case "HT_FT":
		htRes := "X"
		if homeHT > awayHT {
			htRes = "1"
		} else if homeHT < awayHT {
			htRes = "2"
		}

		ftRes := "X"
		if home > away {
			ftRes = "1"
		} else if home < away {
			ftRes = "2"
		}

		actual := htRes + "/" + ftRes
		if sel.Selection == actual || sel.MarketSpec == actual || sel.Selection == (htRes+ftRes) {
			return "WON"
		}
		return "LOST"
	}

	return "PENDING"
}

func parseHandicap(spec string) (float64, float64) {
	if strings.Contains(spec, ":") {
		parts := strings.Split(spec, ":")
		if len(parts) == 2 {
			h, _ := strconv.ParseFloat(parts[0], 64)
			a, _ := strconv.ParseFloat(parts[1], 64)
			return h, a
		}
	}
	v, _ := strconv.ParseFloat(spec, 64)
	return v, 0
}
