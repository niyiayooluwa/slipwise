// Package sportybet implements the SportyBet integration.
package sportybet

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"slipwise/internal/betting/domain"
)

type CloudflareClient interface {
	FetchTicketByCode(ctx context.Context, shareCode string) ([]byte, error)
}

type Provider struct {
	client CloudflareClient
}

func NewProvider(client CloudflareClient) *Provider {
	return &Provider{client: client}
}

func (p *Provider) FetchAndParse(ctx context.Context, shareCode string) (*domain.SlipwiseTicket, error) {
	rawJSON, err := p.client.FetchTicketByCode(ctx, shareCode)
	if err != nil {
		return nil, err
	}

	var payload SportyBetPayload
	if err := json.Unmarshal(rawJSON, &payload); err != nil {
		return nil, err
	}

	matches, bookingSelections := TranslateSportyBet(payload)

	totalOdds := 1.0

	var ticketSelections []domain.TicketSelection
	for _, sel := range bookingSelections {
		totalOdds *= sel.Odds

		var matchDetails domain.MatchDetails
		for _, m := range matches {
			if m.ExternalMatchID == sel.ExternalMatchID {
				matchDetails = domain.MatchDetails{
					HomeTeam:  m.HomeTeam,
					AwayTeam:  m.AwayTeam,
					StartTime: time.UnixMilli(m.StartTime),
				}
				break
			}
		}

		var spec *string
		if sel.MarketSpec != "" {
			s := sel.MarketSpec
			spec = &s
		}

		ticketSelections = append(ticketSelections, domain.TicketSelection{
			Match:           matchDetails,
			ExternalMatchID: sel.ExternalMatchID,
			MarketType:      sel.MarketType,
			MarketSpec:      spec,
			Selection:       sel.Selection,
			Odds:            sel.Odds,
		})
	}

	return &domain.SlipwiseTicket{
		Provider:   "SPORTYBET",
		Code:       shareCode,
		TotalOdds:  totalOdds,
		Selections: ticketSelections,
	}, nil
}

type SportyBetTicketSelection struct {
	EventID   string `json:"eventId"`
	MarketID  string `json:"marketId"`
	Specifier string `json:"specifier"`
	OutcomeID string `json:"outcomeId"`
	Stake     int64  `json:"stake"`
}

type SportyBetTicket struct {
	Selections []SportyBetTicketSelection `json:"selections"`
}

type SportyBetOutcome struct {
	ID   string `json:"id"`
	Desc string `json:"desc"`
	Odds string `json:"odds"`
}

type SportyBetMarket struct {
	ID        string             `json:"id"`
	Desc      string             `json:"desc"`
	Name      string             `json:"name"`
	Specifier string             `json:"specifier"`
	Outcomes  []SportyBetOutcome `json:"outcomes"`
}

type SportyBetItem struct {
	EventID           string            `json:"eventId"`
	HomeTeamName      string            `json:"homeTeamName"`
	AwayTeamName      string            `json:"awayTeamName"`
	EstimateStartTime int64             `json:"estimateStartTime"`
	Markets           []SportyBetMarket `json:"markets"`
}

type SportyBetPayload struct {
	Data struct {
		Ticket   SportyBetTicket `json:"ticket"`
		Outcomes []SportyBetItem `json:"outcomes"`
	} `json:"data"`
}

func TranslateSportyBet(payload SportyBetPayload) ([]domain.Match, []domain.BookingSelection) {
	var matches []domain.Match
	var selections []domain.BookingSelection

	outcomeMap := make(map[string]SportyBetItem)
	for _, item := range payload.Data.Outcomes {
		outcomeMap[item.EventID] = item
	}

	seenMatches := make(map[string]bool)

	for _, sel := range payload.Data.Ticket.Selections {
		item, exists := outcomeMap[sel.EventID]
		if !exists {
			continue
		}

		if !seenMatches[item.EventID] {
			matches = append(matches, domain.Match{
				ExternalMatchID: item.EventID,
				HomeTeam:        item.HomeTeamName,
				AwayTeam:        item.AwayTeamName,
				StartTime:       item.EstimateStartTime,
			})
			seenMatches[item.EventID] = true
		}

		var mType, mSpec, selectionName string
		var odds = 1.0

		for _, market := range item.Markets {
			if market.ID == sel.MarketID {
				marketText := market.Name
				if marketText == "" {
					marketText = market.Desc
				}

				mType = MapMarketType(marketText, item.HomeTeamName, item.AwayTeamName)
				mSpec = CleanSpecifier(market.Specifier)

				for _, outcome := range market.Outcomes {
					if outcome.ID == sel.OutcomeID {
						selectionName = MapSelection(outcome.Desc)
						parsedOdds, err := strconv.ParseFloat(outcome.Odds, 64)
						if err == nil {
							odds = parsedOdds
						}
						break
					}
				}
				break
			}
		}

		selections = append(selections, domain.BookingSelection{
			Provider:        "SPORTYBET",
			ExternalMatchID: sel.EventID,
			MarketType:      mType,
			MarketSpec:      mSpec,
			Selection:       selectionName,
			Odds:            odds,
			Status:          "PENDING",
		})
	}

	return matches, selections
}

func MapMarketType(desc, homeTeam, awayTeam string) string {
	desc = strings.ToLower(desc)

	// Extract Combos first
	if strings.Contains(desc, "&") || strings.Contains(desc, " and ") {
		parts := strings.Split(desc, "&")
		if len(parts) != 2 {
			parts = strings.Split(desc, " and ")
		}
		if len(parts) == 2 {
			m1 := mapBaseMarket(strings.TrimSpace(parts[0]), homeTeam, awayTeam)
			m2 := mapBaseMarket(strings.TrimSpace(parts[1]), homeTeam, awayTeam)
			if m1 != "UNKNOWN" && m2 != "UNKNOWN" {
				return m1 + "_AND_" + m2
			}
		}
	}

	return mapBaseMarket(desc, homeTeam, awayTeam)
}

func mapBaseMarket(desc, homeTeam, awayTeam string) string {
	// Blacklist markets that cannot be mathematically tracked by simple HT/FT goal scores
	lowerDesc := strings.ToLower(desc)
	if strings.Contains(lowerDesc, "early goals") || strings.Contains(lowerDesc, "minute") ||
		strings.Contains(lowerDesc, "corner") || strings.Contains(lowerDesc, "card") ||
		strings.Contains(lowerDesc, "booking") || strings.Contains(lowerDesc, "1up") ||
		strings.Contains(lowerDesc, "2up") || strings.Contains(lowerDesc, "never down") {
		return "UNKNOWN"
	}

	prefix := ""
	if strings.Contains(desc, "1st half") || strings.Contains(desc, "halftime") {
		prefix = "1ST_HALF_"
	} else if strings.Contains(desc, "2nd half") {
		prefix = "2ND_HALF_"
	}

	// Stripping prefixes to make base matching clean
	cleanDesc := strings.ReplaceAll(desc, "1st half - ", "")
	cleanDesc = strings.ReplaceAll(cleanDesc, "1st half ", "")
	cleanDesc = strings.ReplaceAll(cleanDesc, "1st half", "")
	cleanDesc = strings.ReplaceAll(cleanDesc, "2nd half - ", "")
	cleanDesc = strings.ReplaceAll(cleanDesc, "2nd half ", "")
	cleanDesc = strings.ReplaceAll(cleanDesc, "2nd half", "")
	cleanDesc = strings.ReplaceAll(cleanDesc, "halftime", "")

	// Precompute lowercase team names for robust mapping
	homeStr := strings.ToLower(homeTeam)
	awayStr := strings.ToLower(awayTeam)

	base := "UNKNOWN"
	if strings.Contains(cleanDesc, "1x2") {
		base = "MATCH_RESULT"
	} else if strings.Contains(cleanDesc, "asian over/under") {
		base = "ASIAN_OVER_UNDER"
	} else if strings.Contains(cleanDesc, "asian handicap") {
		base = "ASIAN_HANDICAP"
	} else if strings.Contains(cleanDesc, "over/under") || strings.Contains(cleanDesc, "over / under") || strings.Contains(cleanDesc, "o/u") {
		// Use explicit home/away strings or match against the exact team names to map it securely
		if strings.Contains(cleanDesc, "home") || (homeStr != "" && strings.Contains(cleanDesc, homeStr)) {
			base = "HOME_OVER_UNDER"
		} else if strings.Contains(cleanDesc, "away") || (awayStr != "" && strings.Contains(cleanDesc, awayStr)) {
			base = "AWAY_OVER_UNDER"
		} else {
			base = "OVER_UNDER"
		}
	} else if strings.Contains(cleanDesc, "both teams to score") || strings.Contains(cleanDesc, "gg/ng") || strings.Contains(cleanDesc, "goal/no goal") {
		if strings.Contains(cleanDesc, "2+") {
			base = "GG2+"
		} else {
			base = "BTTS"
		}
	} else if strings.Contains(cleanDesc, "double chance") {
		base = "DOUBLE_CHANCE"
	} else if strings.Contains(cleanDesc, "handicap") {
		base = "HANDICAP"
	} else if strings.Contains(cleanDesc, "correct score") {
		base = "CORRECT_SCORE"
	} else if strings.Contains(cleanDesc, "half-time/full-time") || strings.Contains(cleanDesc, "ht/ft") || strings.Contains(cleanDesc, "half time/full time") {
		base = "HT_FT"
	} else if strings.Contains(cleanDesc, "exact goals") {
		base = "EXACT_GOALS"
	} else if strings.Contains(cleanDesc, "teams to score") {
		base = "TEAMS_TO_SCORE"
	} else if strings.Contains(cleanDesc, "no bet") {
		if strings.Contains(cleanDesc, "home") || (homeStr != "" && strings.Contains(cleanDesc, homeStr)) {
			base = "HOME_NO_BET"
		} else if strings.Contains(cleanDesc, "away") || (awayStr != "" && strings.Contains(cleanDesc, awayStr)) {
			base = "AWAY_NO_BET"
		} else if strings.Contains(cleanDesc, "draw") {
			base = "DRAW_NO_BET"
		}
	} else if strings.Contains(cleanDesc, "or over") {
		if strings.Contains(cleanDesc, "home") || (homeStr != "" && strings.Contains(cleanDesc, homeStr)) {
			base = "HOME_OR_OVER"
		} else if strings.Contains(cleanDesc, "away") || (awayStr != "" && strings.Contains(cleanDesc, awayStr)) {
			base = "AWAY_OR_OVER"
		} else if strings.Contains(cleanDesc, "draw") {
			base = "DRAW_OR_OVER"
		}
	} else if strings.Contains(cleanDesc, "or under") {
		if strings.Contains(cleanDesc, "home") || (homeStr != "" && strings.Contains(cleanDesc, homeStr)) {
			base = "HOME_OR_UNDER"
		} else if strings.Contains(cleanDesc, "away") || (awayStr != "" && strings.Contains(cleanDesc, awayStr)) {
			base = "AWAY_OR_UNDER"
		} else if strings.Contains(cleanDesc, "draw") {
			base = "DRAW_OR_UNDER"
		}
	} else if strings.Contains(cleanDesc, "or gg") {
		if strings.Contains(cleanDesc, "home") || (homeStr != "" && strings.Contains(cleanDesc, homeStr)) {
			base = "HOME_OR_GG"
		} else if strings.Contains(cleanDesc, "away") || (awayStr != "" && strings.Contains(cleanDesc, awayStr)) {
			base = "AWAY_OR_GG"
		} else if strings.Contains(cleanDesc, "draw") {
			base = "DRAW_OR_GG"
		}
	} else if strings.Contains(cleanDesc, "or any clean sheet") || strings.Contains(cleanDesc, "or clean sheet") {
		if strings.Contains(cleanDesc, "home") || (homeStr != "" && strings.Contains(cleanDesc, homeStr)) {
			base = "HOME_OR_CLEAN_SHEET"
		} else if strings.Contains(cleanDesc, "away") || (awayStr != "" && strings.Contains(cleanDesc, awayStr)) {
			base = "AWAY_OR_CLEAN_SHEET"
		} else if strings.Contains(cleanDesc, "draw") {
			base = "DRAW_OR_CLEAN_SHEET"
		}
	}

	if base == "UNKNOWN" {
		return "UNKNOWN"
	}
	return prefix + base
}

func CleanSpecifier(spec string) string {
	parts := strings.Split(spec, "=")
	if len(parts) == 2 {
		return parts[1]
	}
	// For combos it might be "total=2.5|total=3.5"
	if strings.Contains(spec, "|") {
		combos := strings.Split(spec, "|")
		res := []string{}
		for _, c := range combos {
			p := strings.Split(c, "=")
			if len(p) == 2 {
				res = append(res, p[1])
			} else {
				res = append(res, c)
			}
		}
		return strings.Join(res, "_AND_")
	}
	return spec
}

func MapSelection(desc string) string {
	if strings.Contains(desc, " & ") || strings.Contains(desc, " and ") {
		parts := strings.Split(desc, " & ")
		if len(parts) != 2 {
			parts = strings.Split(desc, " and ")
		}
		if len(parts) == 2 {
			s1 := mapBaseSelection(strings.TrimSpace(parts[0]))
			s2 := mapBaseSelection(strings.TrimSpace(parts[1]))
			return s1 + "_AND_" + s2
		}
	}
	return mapBaseSelection(desc)
}

func mapBaseSelection(desc string) string {
	lower := strings.ToLower(desc)

	if lower == "1" || lower == "home" {
		return "1"
	} else if lower == "x" || lower == "draw" {
		return "X"
	} else if lower == "2" || lower == "away" {
		return "2"
	} else if strings.Contains(lower, "over") {
		return "OVER"
	} else if strings.Contains(lower, "under") {
		return "UNDER"
	} else if lower == "yes" || lower == "gg" {
		return "YES"
	} else if lower == "no" || lower == "ng" {
		return "NO"
	} else if strings.Contains(lower, "1x") || lower == "home or draw" {
		return "1X"
	} else if strings.Contains(lower, "12") || lower == "home or away" {
		return "12"
	} else if strings.Contains(lower, "x2") || lower == "draw or away" {
		return "X2"
	} else if lower == "only away" {
		return "AWAY_ONLY"
	} else if lower == "only home" {
		return "HOME_ONLY"
	} else if lower == "both" {
		return "BOTH"
	} else if lower == "none" {
		return "NONE"
	}

	if strings.HasPrefix(lower, "home (") {
		return "1"
	}
	if strings.HasPrefix(lower, "draw (") {
		return "X"
	}
	if strings.HasPrefix(lower, "away (") {
		return "2"
	}

	if strings.Contains(lower, "/") {
		res := lower
		res = strings.ReplaceAll(res, "home", "1")
		res = strings.ReplaceAll(res, "draw", "x")
		res = strings.ReplaceAll(res, "away", "2")
		return strings.ToUpper(res)
	}

	return desc
}
