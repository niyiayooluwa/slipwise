package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	db "slipwise/internal/db/generated"

	"github.com/google/uuid"
)

type PendingMatch struct {
	MatchID    uuid.UUID
	ProviderID string
}

// Repository defines the methods required by the poller to interact with the database.
type Repository interface {
	GetPendingMatches(ctx context.Context) ([]PendingMatch, error)
	GetStuckMatches(ctx context.Context) ([]db.GetStuckMatchesRow, error)
}

// Evaluator defines the interface for evaluating matches.
type Evaluator interface {
	Evaluate(ctx context.Context, matchID uuid.UUID, providerID string, data []byte) error
}

// Poller runs as a continuous background daemon that monitors active matches.
//
// Dual-Strategy Engine:
// 1. Firehose Ingestion: Periodically fetches live matches currently broadcasting on SportyBet
//    and updates match clocks / live scores.
// 2. Sweeper Routine: Periodically identifies "stuck" matches (e.g. matches whose kickoff was >3 hours ago
//    but were never marked ENDED due to temporary network blips) and forcibly resolves them.
type Poller struct {
	client    CloudflareClient
	repo      Repository
	evaluator Evaluator
	interval  time.Duration
}

// NewPoller initializes a poller instance.
func NewPoller(client CloudflareClient, repo Repository, evaluator Evaluator, interval time.Duration) *Poller {
	return &Poller{
		client:    client,
		repo:      repo,
		evaluator: evaluator,
		interval:  interval,
	}
}

// Start begins the polling loop.
func (p *Poller) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Poller stopped")
			return
		case <-ticker.C:
			p.RunOnce(ctx)
		}
	}
}

// RunOnce executes a single polling iteration. Exported so it can be triggered by external cron jobs.
func (p *Poller) RunOnce(ctx context.Context) {
	// 1. Fetch pending matches from the database
	pendingMatches, err := p.repo.GetPendingMatches(ctx)
	if err != nil {
		log.Printf("Failed to get pending matches: %v", err)
		return
	}

	if len(pendingMatches) > 0 {
		// 2. Fetch live matches from Cloudflare
		liveMatches, err := p.client.FetchLiveMatches(ctx)
		if err != nil {
			log.Printf("Failed to fetch live matches: %v", err)
		} else {
			// Create a map for quick lookup
			liveMatchMap := make(map[string][]byte)
			for _, match := range liveMatches {
				liveMatchMap[match.ProviderID] = match.Data
			}

			// 3. Loop through pending matches and evaluate if they are live
			for _, pm := range pendingMatches {
				if data, ok := liveMatchMap[pm.ProviderID]; ok {
					if err := p.evaluator.Evaluate(ctx, pm.MatchID, pm.ProviderID, data); err != nil {
						log.Printf("Failed to evaluate live match %s: %v", pm.ProviderID, err)
					}
				}
			}
		}
	}

	// Sweeper: Force-settle matches that disappeared from the live feed without being marked ENDED.
	// These are matches that kicked off 3+ hours ago but are still NOT_STARTED/LIVE in our DB.
	// Since SportyBet drops ended matches from the /live firehose entirely, we never get an
	// explicit "ENDED" signal for them. So we construct a synthetic payload using the last
	// known score and force the evaluator to settle all pending selections.
	stuckMatches, err := p.repo.GetStuckMatches(ctx)
	if err == nil && len(stuckMatches) > 0 {
		for _, sm := range stuckMatches {
			// Build a minimal synthetic SportyBet event payload using the match's last known score.
			// matchStatus "ended" triggers isEnded=true inside the evaluator.
			syntheticPayload := fmt.Sprintf(
				`{"setScore":"%d:%d","matchStatus":"ended","playedSeconds":"90:00"}`,
				sm.HomeScore, sm.AwayScore,
			)
			if err := p.evaluator.Evaluate(ctx, sm.ID, sm.ProviderID, []byte(syntheticPayload)); err != nil {
				log.Printf("Sweeper failed to force-settle stuck match %s: %v", sm.ProviderID, err)
			} else {
				log.Printf("Sweeper force-settled stuck match %s (%d:%d)", sm.ProviderID, sm.HomeScore, sm.AwayScore)
			}
		}
	}
}
