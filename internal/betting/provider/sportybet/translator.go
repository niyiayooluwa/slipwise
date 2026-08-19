// Package sportybet provides translation logic for SportyBet JSON payloads.
package sportybet

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"slipwise/internal/betting/domain"
)

// CloudflareClient is a local interface so the sportybet package does
// not import the worker package directly.
type CloudflareClient interface {
	FetchTicketByCode(ctx context.Context, shareCode string) ([]byte, error)
}

// Provider implements service.TicketProvider for SportyBet.
type Provider struct {
	client CloudflareClient
}

// NewProvider creates a new SportyBet provider.
func NewProvider(client CloudflareClient) *Provider {
	return &Provider{client: client}
}

// FetchAndParse fetches raw JSON via the Cloudflare proxy and translates
// it into a domain.SlipwiseTicket. No DB interaction.
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

// TranslateSportyBet maps SportyBet JSON directly to MVP Database Schema.
func TranslateSportyBet(payload SportyBetPayload) ([]domain.Match, []domain.BookingSelection) {
	var matches []domain.Match
	var selections []domain.BookingSelection

	// Build a lookup map for outcomes (metadata)
	outcomeMap := make(map[string]SportyBetItem)
	for _, item := range payload.Data.Outcomes {
		outcomeMap[item.EventID] = item
	}

	// Keep track of unique matches to avoid duplicates
	seenMatches := make(map[string]bool)

	// Iterate over the ACTUAL selections the user placed
	for _, sel := range payload.Data.Ticket.Selections {
		item, exists := outcomeMap[sel.EventID]
		if !exists {
			continue // Should not happen, but safeguard
		}

		// Add unique matches
		if !seenMatches[item.EventID] {
			matches = append(matches, domain.Match{
				ExternalMatchID: item.EventID,
				HomeTeam:        item.HomeTeamName,
				AwayTeam:        item.AwayTeamName,
				StartTime:       item.EstimateStartTime,
			})
			seenMatches[item.EventID] = true
		}

		// Find the correct market and outcome metadata
		var mType, mSpec, selectionName string
		var odds = 1.0

		for _, market := range item.Markets {
			if market.ID == sel.MarketID {
				// Use Name if available, fallback to Desc
				marketText := market.Name
				if marketText == "" {
					marketText = market.Desc
				}

				mType = MapMarketType(marketText)
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

func MapMarketType(desc string) string {
	desc = strings.ToLower(desc)
	if strings.Contains(desc, "1x2") {
		return "MATCH_RESULT"
	} else if strings.Contains(desc, "asian over/under") {
		return "ASIAN_OVER_UNDER"
	} else if strings.Contains(desc, "asian handicap") {
		return "ASIAN_HANDICAP"
	} else if strings.Contains(desc, "over/under") || strings.Contains(desc, "over / under") {
		if strings.Contains(desc, "home") {
			return "HOME_OVER_UNDER"
		} else if strings.Contains(desc, "away") {
			return "AWAY_OVER_UNDER"
		}
		return "OVER_UNDER"
	} else if strings.Contains(desc, "both teams to score") {
		return "BTTS"
	} else if strings.Contains(desc, "double chance") {
		return "DOUBLE_CHANCE"
	} else if strings.Contains(desc, "draw no bet") {
		return "DRAW_NO_BET"
	} else if strings.Contains(desc, "handicap") {
		return "HANDICAP"
	} else if strings.Contains(desc, "correct score") {
		return "CORRECT_SCORE"
	} else if strings.Contains(desc, "half-time/full-time") || strings.Contains(desc, "ht/ft") {
		return "HT_FT"
	} else if strings.Contains(desc, "home no bet") {
		return "HOME_NO_BET"
	} else if strings.Contains(desc, "away no bet") {
		return "AWAY_NO_BET"
	} else if strings.Contains(desc, "exact goals") {
		return "EXACT_GOALS"
	} else if strings.Contains(desc, "gg2+") || (strings.Contains(desc, "both teams") && strings.Contains(desc, "2")) {
		return "GG2+"
	}
	return "UNKNOWN"
}

func CleanSpecifier(spec string) string {
	parts := strings.Split(spec, "=")
	if len(parts) == 2 {
		return parts[1]
	}
	return spec
}

func MapSelection(desc string) string {
	lower := strings.ToLower(desc)

	// String mapping fixed: Avoid false positives like "Home Over 1.5" mapping to "1"
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
	} else if strings.Contains(lower, "1x") {
		return "1X"
	} else if strings.Contains(lower, "12") {
		return "12"
	} else if strings.Contains(lower, "x2") {
		return "X2"
	}
	// Fallback for correct scores (e.g. 1:0) or formats like "Home (2:0)"
	// Extract the actual 1, X, 2 if it's formatted like "Home (2:0)" for handicap
	if strings.HasPrefix(lower, "home (") {
		return "1"
	}
	if strings.HasPrefix(lower, "draw (") {
		return "X"
	}
	if strings.HasPrefix(lower, "away (") {
		return "2"
	}
	return desc
}
