package service

import (
	"strings"
)

// FormatDisplaySelection transforms technical market enums and codes into clean,
// human-readable labels for the mobile app UI.
//
// Examples:
// - Market: "MATCH_RESULT", Selection: "1", Home: "Arsenal", Away: "Chelsea" -> "Arsenal to Win"
// - Market: "DOUBLE_CHANCE", Selection: "1X", Home: "Arsenal", Away: "Chelsea" -> "Arsenal or Draw"
// - Market: "OVER_UNDER", Spec: "total=2.5", Selection: "OVER" -> "Over 2.5 Goals"
// - Combo Market: "MATCH_RESULT_AND_OVER_UNDER" -> "Arsenal to Win & Over 2.5 Goals"
func FormatDisplaySelection(marketType, marketSpec, selection, homeTeam, awayTeam string) string {
	// Split compound markets if they exist (e.g., MATCH_RESULT_AND_OVER_UNDER)
	if strings.Contains(marketType, "_AND_") && strings.Contains(selection, "_AND_") {
		marketParts := strings.Split(marketType, "_AND_")
		selParts := strings.Split(selection, "_AND_")
		if len(marketParts) == 2 && len(selParts) == 2 {
			part1 := formatSingleMarket(marketParts[0], "", selParts[0], homeTeam, awayTeam)
			part2 := formatSingleMarket(marketParts[1], marketSpec, selParts[1], homeTeam, awayTeam)
			return part1 + " & " + part2
		}
	}

	return formatSingleMarket(marketType, marketSpec, selection, homeTeam, awayTeam)
}

func formatSingleMarket(marketType, marketSpec, selection, homeTeam, awayTeam string) string {
	// First, resolve the base selection text (e.g. 1 -> Home Team Name)
	baseSel := selection
	switch selection {
	case "1":
		baseSel = homeTeam
	case "X":
		baseSel = "Draw"
	case "2":
		baseSel = awayTeam
	case "1X":
		baseSel = homeTeam + " or Draw"
	case "12":
		baseSel = homeTeam + " or " + awayTeam
	case "X2":
		baseSel = "Draw or " + awayTeam
	case "YES":
		baseSel = "Yes"
	case "NO":
		baseSel = "No"
	case "OVER":
		baseSel = "Over"
	case "UNDER":
		baseSel = "Under"
	}

	// Then, combine it with the market logic
	switch marketType {
	case "MATCH_RESULT":
		if selection == "X" {
			return "Draw"
		}
		return baseSel + " to Win"
	case "1ST_HALF_MATCH_RESULT":
		if selection == "X" {
			return "Draw (1st Half)"
		}
		return baseSel + " to Win 1st Half"
	case "2ND_HALF_MATCH_RESULT":
		if selection == "X" {
			return "Draw (2nd Half)"
		}
		return baseSel + " to Win 2nd Half"

	case "DOUBLE_CHANCE":
		return baseSel
	case "1ST_HALF_DOUBLE_CHANCE":
		return baseSel + " (1st Half)"
	case "2ND_HALF_DOUBLE_CHANCE":
		return baseSel + " (2nd Half)"

	case "DRAW_NO_BET", "HOME_NO_BET", "AWAY_NO_BET":
		if selection == "X" {
			return "Draw"
		}
		return baseSel + " to Win (DNB)"

	case "OVER_UNDER":
		spec := parseSpecifierTotal(marketSpec)
		return baseSel + " " + spec + " Goals"
	case "1ST_HALF_OVER_UNDER":
		spec := parseSpecifierTotal(marketSpec)
		return "1st Half " + baseSel + " " + spec + " Goals"
	case "2ND_HALF_OVER_UNDER":
		spec := parseSpecifierTotal(marketSpec)
		return "2nd Half " + baseSel + " " + spec + " Goals"

	case "HOME_OVER_UNDER":
		spec := parseSpecifierTotal(marketSpec)
		return homeTeam + " " + baseSel + " " + spec + " Goals"

	case "AWAY_OVER_UNDER":
		spec := parseSpecifierTotal(marketSpec)
		return awayTeam + " " + baseSel + " " + spec + " Goals"

	case "HOME_OR_OVER":
		spec := parseSpecifierTotal(marketSpec)
		return homeTeam + " to Win or Over " + spec + " Goals"
	case "AWAY_OR_OVER":
		spec := parseSpecifierTotal(marketSpec)
		return awayTeam + " to Win or Over " + spec + " Goals"
	case "DRAW_OR_OVER":
		spec := parseSpecifierTotal(marketSpec)
		return "Draw or Over " + spec + " Goals"

	case "HOME_OR_UNDER":
		spec := parseSpecifierTotal(marketSpec)
		return homeTeam + " to Win or Under " + spec + " Goals"
	case "AWAY_OR_UNDER":
		spec := parseSpecifierTotal(marketSpec)
		return awayTeam + " to Win or Under " + spec + " Goals"
	case "DRAW_OR_UNDER":
		spec := parseSpecifierTotal(marketSpec)
		return "Draw or Under " + spec + " Goals"

	case "HOME_OR_GG":
		return homeTeam + " to Win or Both Teams to Score"
	case "AWAY_OR_GG":
		return awayTeam + " to Win or Both Teams to Score"
	case "DRAW_OR_GG":
		return "Draw or Both Teams to Score"

	case "HOME_OR_CLEAN_SHEET":
		return homeTeam + " to Win or Clean Sheet"
	case "AWAY_OR_CLEAN_SHEET":
		return awayTeam + " to Win or Clean Sheet"
	case "DRAW_OR_CLEAN_SHEET":
		return "Draw or Clean Sheet"

	case "ASIAN_OVER_UNDER", "ASIAN_HANDICAP", "HANDICAP":
		spec := parseSpecifierTotal(marketSpec)
		return baseSel + " (" + spec + ")"

	case "BTTS":
		if selection == "YES" {
			return "Both Teams to Score"
		}
		return "Both Teams to Score: No"
	case "1ST_HALF_BTTS":
		if selection == "YES" {
			return "Both Teams to Score in 1st Half"
		}
		return "Both Teams to Score in 1st Half: No"
	case "2ND_HALF_BTTS":
		if selection == "YES" {
			return "Both Teams to Score in 2nd Half"
		}
		return "Both Teams to Score in 2nd Half: No"

	case "CORRECT_SCORE":
		return "Correct Score: " + selection

	case "HT_FT":
		parts := strings.Split(selection, "/")
		if len(parts) == 2 {
			ht := parts[0]
			ft := parts[1]
			if ht == "1" {
				ht = homeTeam
			} else if ht == "X" {
				ht = "Draw"
			} else if ht == "2" {
				ht = awayTeam
			}
			if ft == "1" {
				ft = homeTeam
			} else if ft == "X" {
				ft = "Draw"
			} else if ft == "2" {
				ft = awayTeam
			}
			return ht + " / " + ft
		}
		return selection

	case "UNKNOWN":
		return "Unknown Market (" + selection + ")"
	}

	// Fallback for unmapped markets: strip underscores, clean title case
	words := strings.Split(strings.ReplaceAll(marketType, "_", " "), " ")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	cleanedMarket := strings.Join(words, " ")
	if baseSel != "" && baseSel != selection {
		return cleanedMarket + ": " + baseSel
	} else if baseSel != "" {
		return cleanedMarket + " (" + baseSel + ")"
	}
	return cleanedMarket
}

func parseSpecifierTotal(spec string) string {
	if spec == "" {
		return ""
	}
	parts := strings.Split(spec, "|")
	for _, p := range parts {
		if strings.HasPrefix(p, "total=") {
			return strings.TrimPrefix(p, "total=")
		}
	}
	return spec
}
