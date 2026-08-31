// Package service houses the core betting business logic, including the
// Mathematical Evaluation Engine (evaluator.go) and the Selection Formatter (formatter.go).
package service

import (
	"fmt"
	"strconv"
	"strings"

	"slipwise/internal/betting/domain"
)

// MatchScore carries the full score progression of a football match.
type MatchScore struct {
	HomeScoreHT int // Half-time score (e.g. 1)
	AwayScoreHT int // Half-time score (e.g. 0)
	HomeScoreFT int // Full-time final score (e.g. 2)
	AwayScoreFT int // Full-time final score (e.g. 1)
}

// getPhaseScores extracts the relevant score slice depending on the market's phase.
// For example:
// - "1ST_HALF_OVER_UNDER" uses HT scores.
// - "2ND_HALF_MATCH_RESULT" computes the 2nd half delta: (FT - HT).
// - Regular full-time markets use FT scores.
func getPhaseScores(marketType string, score MatchScore) (int, int, int) {
	if strings.HasPrefix(marketType, "1ST_HALF_") {
		return score.HomeScoreHT, score.AwayScoreHT, score.HomeScoreHT + score.AwayScoreHT
	}
	if strings.HasPrefix(marketType, "2ND_HALF_") {
		h2 := score.HomeScoreFT - score.HomeScoreHT
		a2 := score.AwayScoreFT - score.AwayScoreHT
		return h2, a2, h2 + a2
	}
	return score.HomeScoreFT, score.AwayScoreFT, score.HomeScoreFT + score.AwayScoreFT
}

// stripPhasePrefix removes the temporal prefix to reuse standard full-time market math
// for 1st/2nd half variations.
func stripPhasePrefix(marketType string) string {
	m := strings.TrimPrefix(marketType, "1ST_HALF_")
	m = strings.TrimPrefix(m, "2ND_HALF_")
	return m
}

// EvaluateSelection is the pure mathematical core of the betting platform.
// Given a user's bet selection and the actual match score, it returns:
// "WON", "LOST", "VOID", "HALF_WON", or "HALF_LOST".
//
// Note: This function computes pure mathematical truth.
// The caller (live_evaluator.go) is responsible for applying live match failsafes.
func EvaluateSelection(sel domain.BookingSelection, score MatchScore) string {
	if strings.Contains(sel.MarketType, "_AND_") {
		parts := strings.Split(sel.MarketType, "_AND_")
		selParts := strings.Split(sel.Selection, "_AND_")
		// Safely split specs if there are multiple, otherwise pass the same one down
		specParts := strings.Split(sel.MarketSpec, "_AND_")

		if len(parts) == 2 && len(selParts) == 2 {
			spec1 := sel.MarketSpec
			spec2 := sel.MarketSpec
			if len(specParts) == 2 {
				spec1 = specParts[0]
				spec2 = specParts[1]
			}

			res1 := EvaluateSelection(domain.BookingSelection{
				MarketType: parts[0],
				MarketSpec: spec1,
				Selection:  selParts[0],
			}, score)
			res2 := EvaluateSelection(domain.BookingSelection{
				MarketType: parts[1],
				MarketSpec: spec2,
				Selection:  selParts[1],
			}, score)

			if res1 == "WON" && res2 == "WON" {
				return "WON"
			}
			if res1 == "LOST" || res2 == "LOST" {
				return "LOST"
			}
			return "PENDING"
		}
	}

	home, away, total := getPhaseScores(sel.MarketType, score)
	baseMarket := stripPhasePrefix(sel.MarketType)

	switch baseMarket {
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
		if home >= away && sel.Selection == "1X" {
			return "WON"
		}
		if home != away && sel.Selection == "12" {
			return "WON"
		}
		if home <= away && sel.Selection == "X2" {
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
		if sel.Selection == actual || sel.MarketSpec == actual || strings.Contains(sel.Selection, actual) || strings.Contains(sel.MarketSpec, actual) {
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
		diff := float64(home) + hcp - float64(away)
		if sel.Selection == "2" {
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
		if sel.Selection == expected || sel.MarketSpec == expected || sel.Selection == strconv.Itoa(total) || sel.MarketSpec == strconv.Itoa(total) {
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
		if score.HomeScoreHT > score.AwayScoreHT {
			htRes = "1"
		} else if score.HomeScoreHT < score.AwayScoreHT {
			htRes = "2"
		}
		ftRes := "X"
		if score.HomeScoreFT > score.AwayScoreFT {
			ftRes = "1"
		} else if score.HomeScoreFT < score.AwayScoreFT {
			ftRes = "2"
		}

		actual := htRes + "/" + ftRes
		if sel.Selection == actual || sel.MarketSpec == actual || sel.Selection == (htRes+ftRes) {
			return "WON"
		}
		return "LOST"

	case "TEAMS_TO_SCORE":
		switch sel.Selection {
		case "HOME_ONLY", "ONLY_HOME":
			if home > 0 && away == 0 {
				return "WON"
			}
		case "AWAY_ONLY", "ONLY_AWAY":
			if away > 0 && home == 0 {
				return "WON"
			}
		case "BOTH":
			if home > 0 && away > 0 {
				return "WON"
			}
		case "NONE":
			if home == 0 && away == 0 {
				return "WON"
			}
		}
		return "LOST"

	case "HOME_OR_OVER", "DRAW_OR_OVER", "AWAY_OR_OVER":
		spec, _ := strconv.ParseFloat(sel.MarketSpec, 64)
		cond1 := false
		if baseMarket == "HOME_OR_OVER" {
			cond1 = home > away
		}
		if baseMarket == "DRAW_OR_OVER" {
			cond1 = home == away
		}
		if baseMarket == "AWAY_OR_OVER" {
			cond1 = home < away
		}
		cond2 := float64(total) > spec

		if cond1 || cond2 {
			if sel.Selection == "YES" {
				return "WON"
			} else {
				return "LOST"
			}
		} else {
			if sel.Selection == "YES" {
				return "LOST"
			} else {
				return "WON"
			}
		}

	case "HOME_OR_UNDER", "DRAW_OR_UNDER", "AWAY_OR_UNDER":
		spec, _ := strconv.ParseFloat(sel.MarketSpec, 64)
		cond1 := false
		if baseMarket == "HOME_OR_UNDER" {
			cond1 = home > away
		}
		if baseMarket == "DRAW_OR_UNDER" {
			cond1 = home == away
		}
		if baseMarket == "AWAY_OR_UNDER" {
			cond1 = home < away
		}
		cond2 := float64(total) < spec

		if cond1 || cond2 {
			if sel.Selection == "YES" {
				return "WON"
			} else {
				return "LOST"
			}
		} else {
			if sel.Selection == "YES" {
				return "LOST"
			} else {
				return "WON"
			}
		}

	case "HOME_OR_GG", "DRAW_OR_GG", "AWAY_OR_GG":
		cond1 := false
		if baseMarket == "HOME_OR_GG" {
			cond1 = home > away
		}
		if baseMarket == "DRAW_OR_GG" {
			cond1 = home == away
		}
		if baseMarket == "AWAY_OR_GG" {
			cond1 = home < away
		}
		cond2 := home > 0 && away > 0

		if cond1 || cond2 {
			if sel.Selection == "YES" {
				return "WON"
			} else {
				return "LOST"
			}
		} else {
			if sel.Selection == "YES" {
				return "LOST"
			} else {
				return "WON"
			}
		}

	case "HOME_OR_CLEAN_SHEET", "DRAW_OR_CLEAN_SHEET", "AWAY_OR_CLEAN_SHEET":
		cond1 := false
		if baseMarket == "HOME_OR_CLEAN_SHEET" {
			cond1 = home > away
		}
		if baseMarket == "DRAW_OR_CLEAN_SHEET" {
			cond1 = home == away
		}
		if baseMarket == "AWAY_OR_CLEAN_SHEET" {
			cond1 = home < away
		}
		cond2 := home == 0 || away == 0

		if cond1 || cond2 {
			if sel.Selection == "YES" {
				return "WON"
			} else {
				return "LOST"
			}
		} else {
			if sel.Selection == "YES" {
				return "LOST"
			} else {
				return "WON"
			}
		}
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
