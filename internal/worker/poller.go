package worker

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
)

type PendingMatch struct {
	MatchID    uuid.UUID
	ProviderID string
}

// Repository defines the methods required by the poller to interact with the database.
type Repository interface {
	GetPendingMatches(ctx context.Context) ([]PendingMatch, error)
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
			p.tick(ctx)
		}
	}
}

func (p *Poller) tick(ctx context.Context) {
	// 1. Fetch pending matches from the database
	pendingMatches, err := p.repo.GetPendingMatches(ctx)
	if err != nil {
		log.Printf("Failed to get pending matches: %v", err)
		return
	}

	if len(pendingMatches) == 0 {
		return // Nothing to do
	}

	// 2. Fetch live matches from Cloudflare
	liveMatches, err := p.client.FetchLiveMatches(ctx)
	if err != nil {
		log.Printf("Failed to fetch live matches: %v", err)
		return
	}

	// Create a map for quick lookup
	liveMatchMap := make(map[string][]byte)
	for _, match := range liveMatches {
		liveMatchMap[match.ProviderID] = match.Data
	}

	// 3. Loop through pending matches and evaluate if they are live
	for _, pm := range pendingMatches {
		if data, ok := liveMatchMap[pm.ProviderID]; ok {
			if err := p.evaluator.Evaluate(ctx, pm.MatchID, pm.ProviderID, data); err != nil {
				log.Printf("Failed to evaluate match %s: %v", pm.ProviderID, err)
			}
		}
	}
}
