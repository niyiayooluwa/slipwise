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
		SetScore    string `json:"setScore"`
		MatchStatus string `json:"matchStatus"`
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

	// Group tokens by user/message to avoid spamming
	// For MVP, we send directly.
	for _, res := range evalResults {
		var title, body string

		if res.TicketStatus == "WON" {
			title = "Ticket Won! 💸🚀"
			body = fmt.Sprintf("Your ticket %s just hit! All %d legs are green.", res.BookingCode, res.TotalLegs)
		} else if res.TicketStatus == "LOST" {
			title = "Ticket Lost ❌"
			body = fmt.Sprintf("Your ticket %s was busted.", res.BookingCode)
		} else if res.PendingLegs == 1 && res.LostLegs == 0 {
			title = "Sweat Alert! 😰"
			body = fmt.Sprintf("Only ONE leg left to win ticket %s! Cash out or pray?", res.BookingCode)
		} else if res.WonLegs > 0 {
			title = "Leg Secured! ✅"
			body = fmt.Sprintf("Progress on ticket %s: %d/%d won.", res.BookingCode, res.WonLegs, res.TotalLegs)
		}

		if title != "" && res.FcmToken != "" {
			_ = e.fcm.SendMulticast(context.Background(), []string{res.FcmToken}, title, body, map[string]string{
				"ticket_id": res.BookingCodeID.String(),
				"type":      "ticket_update",
			})
		}
	}

	return nil
}
