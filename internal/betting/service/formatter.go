package service

import (
	"strings"
)

// FormatDisplaySelection generates a user-friendly string for the frontend UI.
func FormatDisplaySelection(marketType, marketSpec, selection, homeTeam, awayTeam string) string {
	// Split combos if they exist (e.g., MATCH_RESULT_AND_OVER_UNDER)
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
	case "MATCH_RESULT", "1ST_HALF_MATCH_RESULT", "2ND_HALF_MATCH_RESULT":
		if selection == "X" {
			return "Draw"
		}
		return baseSel + " to Win"
	
	case "DOUBLE_CHANCE", "1ST_HALF_DOUBLE_CHANCE", "2ND_HALF_DOUBLE_CHANCE":
		return baseSel
		
	case "DRAW_NO_BET", "HOME_NO_BET", "AWAY_NO_BET":
		if selection == "X" {
			return "Draw"
		}
		return baseSel + " to Win (DNB)"
		
	case "OVER_UNDER", "1ST_HALF_OVER_UNDER", "2ND_HALF_OVER_UNDER":
		spec := parseSpecifierTotal(marketSpec)
		return baseSel + " " + spec + " Goals"
		
	case "HOME_OVER_UNDER":
		spec := parseSpecifierTotal(marketSpec)
		return homeTeam + " " + baseSel + " " + spec + " Goals"
		
	case "AWAY_OVER_UNDER":
		spec := parseSpecifierTotal(marketSpec)
		return awayTeam + " " + baseSel + " " + spec + " Goals"
		
	case "ASIAN_OVER_UNDER", "ASIAN_HANDICAP", "HANDICAP":
		spec := parseSpecifierTotal(marketSpec)
		return baseSel + " (" + spec + ")"
		
	case "BTTS", "1ST_HALF_BTTS", "2ND_HALF_BTTS":
		if selection == "YES" {
			return "Both Teams to Score"
		}
		return "Both Teams to Score: No"
		
	case "CORRECT_SCORE":
		return "Correct Score: " + selection
		
	case "HT_FT":
		parts := strings.Split(selection, "/")
		if len(parts) == 2 {
			ht := parts[0]
			ft := parts[1]
			if ht == "1" { ht = homeTeam } else if ht == "X" { ht = "Draw" } else if ht == "2" { ht = awayTeam }
			if ft == "1" { ft = homeTeam } else if ft == "X" { ft = "Draw" } else if ft == "2" { ft = awayTeam }
			return ht + " / " + ft
		}
		return selection

	case "UNKNOWN":
		return "Unknown Market (" + selection + ")"
	}

	// Fallback for missing mapping
	return marketType + ": " + baseSel
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
