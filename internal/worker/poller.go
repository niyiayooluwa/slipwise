package worker

import (
	"context"
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

// Poller runs in the background and polls for live matches.
type Poller struct {
	client    CloudflareClient
	repo      Repository
	evaluator Evaluator
	interval  time.Duration
}

// NewPoller creates a new Poller.
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

	// 4. Sweeper: Fetch up to 5 stuck matches (older than 3 hours) and force check them.
	stuckMatches, err := p.repo.GetStuckMatches(ctx)
	if err == nil && len(stuckMatches) > 0 {
		for _, sm := range stuckMatches {
			data, err := p.client.FetchTicketByCode(ctx, sm.ProviderID)
			if err != nil {
				log.Printf("Sweeper failed to fetch stuck match %s: %v", sm.ProviderID, err)
				continue
			}
			if err := p.evaluator.Evaluate(ctx, sm.ID, sm.ProviderID, data); err != nil {
				log.Printf("Sweeper failed to evaluate match %s: %v", sm.ProviderID, err)
			}
		}
	}
}
