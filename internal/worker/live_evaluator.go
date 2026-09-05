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
	GetMatchByID(ctx context.Context, id uuid.UUID) (db.GetMatchByIDRow, error)
	GetPendingSelectionsForMatch(ctx context.Context, matchID uuid.UUID) ([]db.GetPendingSelectionsForMatchRow, error)
	SetEarlyWinNotified(ctx context.Context, arg db.SetEarlyWinNotifiedParams) error
	SetHTNotified(ctx context.Context, arg db.SetHTNotifiedParams) error
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
		Status        int    `json:"status"`        // Integer status (e.g. 1=Live, 2/3=Ended)
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

	// 1. Fetch old match state to calculate deltas
	oldMatch, err := e.repo.GetMatchByID(ctx, matchID)
	if err != nil {
		return fmt.Errorf("failed to fetch old match state: %w", err)
	}

	dbMatchStatus := "LIVE"

	// Check both string MatchStatus and integer Status
	isEnded := strings.Contains(strings.ToLower(event.MatchStatus), "end") ||
		strings.Contains(strings.ToLower(event.MatchStatus), "finish") ||
		event.MatchStatus == "2" || event.MatchStatus == "3" ||
		event.Status == 2 || event.Status == 3 || event.Status == 4

	if isEnded {
		dbMatchStatus = "ENDED"
	}

	// Calculate Deltas
	justStarted := oldMatch.Status == "NOT_STARTED" && dbMatchStatus == "LIVE"

	// Assuming H1, HT, H2 in SportyBet. H1->HT or HT->H2
	wasHT := false
	if oldMatch.LiveTime != nil {
		wasHT = strings.Contains(strings.ToLower(*oldMatch.LiveTime), "ht")
	}
	justHitHT := !wasHT && strings.Contains(strings.ToLower(event.MatchStatus), "ht")

	oldTotal := int(oldMatch.HomeScore + oldMatch.AwayScore)
	newTotal := home + away
	goalScored := newTotal > oldTotal
	goalCancelled := newTotal < oldTotal

	// 2. Update match live state in DB

	err = e.repo.UpdateMatchState(ctx, db.UpdateMatchStateParams{
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

	// 2. Process Delta Notifications for individual tickets
	if justStarted || justHitHT || goalScored || goalCancelled {
		pendingSels, err := e.repo.GetPendingSelectionsForMatch(ctx, matchID)
		if err == nil {
			for _, ps := range pendingSels {
				if ps.FcmToken == nil {
					continue
				}

				// --- MUTE GUARD: if the user's ticket is already LOST, stop sending ---
				// GetPendingSelectionsForMatch only returns selections whose status = PENDING,
				// but the parent booking_code may already be LOST (another leg cut it).
				// We skip all hype notifications in that case.
				if ps.BookingCodeStatus == "LOST" {
					continue
				}

				sel := domain.BookingSelection{
					MarketType: ps.MarketType,
					Selection:  ps.Selection,
				}
				if ps.MarketSpec != nil {
					sel.MarketSpec = *ps.MarketSpec
				}

				if goalCancelled {
					title, body, img := notification.GetVARMessage(oldMatch.HomeTeam, oldMatch.AwayTeam)
					e.fcm.SendMulticast(ctx, []string{*ps.FcmToken}, title, body, map[string]string{"ticket_id": ps.BookingCodeID.String(), "type": "ticket_update", "image": img})
					if ps.NotifiedEarlyWin {
						e.repo.SetEarlyWinNotified(ctx, db.SetEarlyWinNotifiedParams{ID: ps.ID, NotifiedEarlyWin: false})
					}
					continue
				}

				if justStarted {
					title, body, img := notification.GetStartMessage(oldMatch.HomeTeam, oldMatch.AwayTeam)
					e.fcm.SendMulticast(ctx, []string{*ps.FcmToken}, title, body, map[string]string{"ticket_id": ps.BookingCodeID.String(), "type": "ticket_update", "image": img})
				}

				if justHitHT && !ps.NotifiedHt {
					status := bettingservice.EvaluateSelection(sel, score)
					title, body, img := notification.GetHTMessage(oldMatch.HomeTeam, oldMatch.AwayTeam, status)
					e.fcm.SendMulticast(ctx, []string{*ps.FcmToken}, title, body, map[string]string{"ticket_id": ps.BookingCodeID.String(), "type": "ticket_update", "image": img})
					e.repo.SetHTNotified(ctx, db.SetHTNotifiedParams{ID: ps.ID, NotifiedHt: true})
				}

				if goalScored && !ps.NotifiedEarlyWin {
					status := bettingservice.EvaluateSelection(sel, score)
					if status == "WON" {
						// Only hype certain markets early
						if strings.Contains(ps.MarketType, "OVER") || strings.Contains(ps.MarketType, "BTTS") || strings.Contains(ps.MarketType, "GG") {

							selDesc := ps.Selection
							if ps.MarketSpec != nil && *ps.MarketSpec != "" {
								selDesc = fmt.Sprintf("%s %s", ps.Selection, *ps.MarketSpec)
							}
							if strings.Contains(ps.MarketType, "OVER") {
								selDesc += " Goals"
								if ps.MarketType == "HOME_OVER_UNDER" {
									selDesc = fmt.Sprintf("%s (Home) %s", oldMatch.HomeTeam, selDesc)
								} else if ps.MarketType == "AWAY_OVER_UNDER" {
									selDesc = fmt.Sprintf("%s (Away) %s", oldMatch.AwayTeam, selDesc)
								}
							} else if strings.Contains(ps.MarketType, "BTTS") || strings.Contains(ps.MarketType, "GG") {
								selDesc = "GG (Both Teams to Score)"
							}

							matchDesc := fmt.Sprintf("%s %d-%d %s", oldMatch.HomeTeam, home, away, oldMatch.AwayTeam)

							title, body, img := notification.GetEarlyHitMessage(selDesc, matchDesc)
							e.fcm.SendMulticast(ctx, []string{*ps.FcmToken}, title, body, map[string]string{"ticket_id": ps.BookingCodeID.String(), "type": "ticket_update", "image": img})
							e.repo.SetEarlyWinNotified(ctx, db.SetEarlyWinNotifiedParams{ID: ps.ID, NotifiedEarlyWin: true})
						}
					}
				}
			}
		}
	}

	// 3. Fetch the pending selections (buckets) for this match (For final settlement)
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
		Image  string
	}
	notifications := make(map[pushMsg][]string)

	for _, res := range evalResults {
		var title, body, img string

		if res.TicketStatus == "WON" {
			title, body, img = notification.GetTicketWinMessage(int(res.TotalLegs))
		} else if res.TicketStatus == "LOST" {
			title, body, img = notification.GetTicketLossMessage(res.BookingCode)
		}
		// "Sweat Alert" and "Leg Secured" notifications are temporarily disabled
		// until Fast Settlement is re-introduced in a future phase.

		// Only queue valid tokens (from the LEFT JOIN)
		if title != "" && res.FcmToken != nil {
			msg := pushMsg{Title: title, Body: body, Ticket: res.BookingCodeID.String(), Image: img}
			notifications[msg] = append(notifications[msg], *res.FcmToken)
		}
	}

	// Dispatch batched notifications
	for msg, tokens := range notifications {
		err := e.fcm.SendMulticast(context.Background(), tokens, msg.Title, msg.Body, map[string]string{
			"ticket_id": msg.Ticket,
			"type":      "ticket_update",
			"image":     msg.Image,
		})
		if err != nil {
			log.Printf("FCM Multicast error for ticket %s: %v", msg.Ticket, err)
		}
	}

	return nil
}
