package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"slipwise/internal/betting/domain"
	bettingservice "slipwise/internal/betting/service"
	db "slipwise/internal/db/generated"
	"slipwise/internal/notification"

	"github.com/google/uuid"
)

type EvaluatorRepo interface {
	GetPendingBucketsForMatch(ctx context.Context, matchID uuid.UUID) ([]db.GetPendingBucketsForMatchRow, error)
	UpdateSelectionStatus(ctx context.Context, arg db.UpdateSelectionStatusParams) ([]uuid.UUID, error)
	EvaluateTickets(ctx context.Context, bookingCodeIds []uuid.UUID) ([]db.EvaluateTicketsRow, error)
	UpdateMatchState(ctx context.Context, arg db.UpdateMatchStateParams) error
}

type liveEvaluator struct {
	repo EvaluatorRepo
	fcm  notification.Service
}

func NewLiveEvaluator(repo EvaluatorRepo, fcm notification.Service) Evaluator {
	return &liveEvaluator{repo: repo, fcm: fcm}
}

func (e *liveEvaluator) Evaluate(ctx context.Context, matchID uuid.UUID, providerID string, data []byte) error {
	var event struct {
		SetScore      string `json:"setScore"`
		MatchStatus   string `json:"matchStatus"`
		PlayedSeconds string `json:"playedSeconds"` // Bug fix: SportyBet uses "playedSeconds", NOT "matchTime"
	}
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	var home, away int
	parts := strings.Split(event.SetScore, ":")
	if len(parts) == 2 {
		home, _ = strconv.Atoi(parts[0])
		away, _ = strconv.Atoi(parts[1])
	}

	// 1. Update match live state in DB
	dbMatchStatus := "LIVE"
	isEnded := strings.Contains(strings.ToLower(event.MatchStatus), "end") || strings.Contains(strings.ToLower(event.MatchStatus), "finish") || event.MatchStatus == "2" || event.MatchStatus == "3"
	if isEnded {
		dbMatchStatus = "ENDED"
	}

	err := e.repo.UpdateMatchState(ctx, db.UpdateMatchStateParams{
		HomeScore: int32(home),
		AwayScore: int32(away),
		Status:    dbMatchStatus,
		LiveTime:  &event.PlayedSeconds,
		ID:        matchID,
	})
	if err != nil {
		log.Printf("Failed to update match state for %s: %v", providerID, err)
	}

	score := bettingservice.MatchScore{
		HomeScoreFT: home,
		AwayScoreFT: away,
		HomeScoreHT: home, // stubbed for MVP
		AwayScoreHT: away, // stubbed for MVP
	}

	// 2. Fetch the pending selections (buckets) for this match
	buckets, err := e.repo.GetPendingBucketsForMatch(ctx, matchID)
	if err != nil {
		return fmt.Errorf("failed to get pending buckets: %w", err)
	}

	var affectedTickets []uuid.UUID

	// 3. Evaluate each bucket
	for _, b := range buckets {
		sel := domain.BookingSelection{
			MarketType: b.MarketType,
			Selection:  b.Selection,
		}
		if b.MarketSpec != nil {
			sel.MarketSpec = *b.MarketSpec
		}

		status := bettingservice.EvaluateSelection(sel, score)

		// MVP Tech Debt: Disable Fast Settlement.
		// To prevent False Wins (e.g. Double Chance at 0-0) and the VAR Problem
		// (goals being cancelled after an Over is settled), we force all bets 
		// to remain PENDING until the final whistle.
		if !isEnded {
			status = "PENDING"
		}

		if status != "PENDING" {
			// Fast settle the bucket!
			ticketIDs, err := e.repo.UpdateSelectionStatus(ctx, db.UpdateSelectionStatusParams{
				Status:     status,
				MatchID:    matchID,
				MarketType: b.MarketType,
				Selection:  b.Selection,
			})
			if err != nil {
				log.Printf("Failed to update status for match %s bucket %s:%s to %s: %v", providerID, b.MarketType, b.Selection, status, err)
			} else {
				affectedTickets = append(affectedTickets, ticketIDs...)
				log.Printf("Settled match %s bucket %s:%s as %s (affected %d tickets)", providerID, b.MarketType, b.Selection, status, len(ticketIDs))
			}
		}
	}

	if len(affectedTickets) == 0 {
		return nil
	}

	// Deduplicate affected tickets
	ticketSet := make(map[uuid.UUID]bool)
	var uniqueTickets []uuid.UUID
	for _, id := range affectedTickets {
		if !ticketSet[id] {
			ticketSet[id] = true
			uniqueTickets = append(uniqueTickets, id)
		}
	}

	// 4. Evaluate Tickets and dispatch Push Notifications
	evalResults, err := e.repo.EvaluateTickets(ctx, uniqueTickets)
	if err != nil {
		return fmt.Errorf("failed to evaluate tickets: %w", err)
	}

	// Group tokens by notification message to batch FCM multicast
	type pushMsg struct {
		Title  string
		Body   string
		Ticket string
	}
	notifications := make(map[pushMsg][]string)

	for _, res := range evalResults {
		var title, body string

		if res.TicketStatus == "WON" {
			title = "You won! 🎉"
			body = fmt.Sprintf("All %d legs landed on ticket %s. Cash out and celebrate! 💸", res.TotalLegs, res.BookingCode)
		} else if res.TicketStatus == "LOST" {
			title = "Better luck next time 😔"
			body = fmt.Sprintf("Ticket %s didn't make it this time. Review it and come back stronger.", res.BookingCode)
		}
		// "Sweat Alert" and "Leg Secured" notifications are temporarily disabled
		// until Fast Settlement is re-introduced in a future phase.

		// Only queue valid tokens (from the LEFT JOIN)
		if title != "" && res.FcmToken != nil {
			msg := pushMsg{Title: title, Body: body, Ticket: res.BookingCodeID.String()}
			notifications[msg] = append(notifications[msg], *res.FcmToken)
		}
	}

	// Dispatch batched notifications
	for msg, tokens := range notifications {
		err := e.fcm.SendMulticast(context.Background(), tokens, msg.Title, msg.Body, map[string]string{
			"ticket_id": msg.Ticket,
			"type":      "ticket_update",
		})
		if err != nil {
			log.Printf("FCM Multicast error for ticket %s: %v", msg.Ticket, err)
		}
	}

	return nil
}
