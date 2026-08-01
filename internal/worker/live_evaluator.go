package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"sportloga/internal/betting/domain"
	bettingservice "sportloga/internal/betting/service"
	db "sportloga/internal/db/generated"

	"github.com/google/uuid"
)

type EvaluatorRepo interface {
	GetPendingBucketsForMatch(ctx context.Context, matchID uuid.UUID) ([]db.GetPendingBucketsForMatchRow, error)
	UpdateSelectionStatus(ctx context.Context, arg db.UpdateSelectionStatusParams) ([]uuid.UUID, error)
}

type liveEvaluator struct {
	repo EvaluatorRepo
}

func NewLiveEvaluator(repo EvaluatorRepo) Evaluator {
	return &liveEvaluator{repo: repo}
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
			// This will settle EVERY ticket that has this selection on this match.
			_, err := e.repo.UpdateSelectionStatus(ctx, db.UpdateSelectionStatusParams{
				Status:     status,
				MatchID:    matchID,
				MarketType: b.MarketType,
				Selection:  b.Selection,
			})
			if err != nil {
				log.Printf("Failed to update status for match %s bucket %s:%s to %s: %v", providerID, b.MarketType, b.Selection, status, err)
			} else {
				log.Printf("Settled match %s bucket %s:%s as %s", providerID, b.MarketType, b.Selection, status)
			}
		}
	}

	return nil
}
