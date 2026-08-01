package worker

import (
	"context"
	"log"
	"time"
)

// Repository defines the methods required by the poller to interact with the database.
type Repository interface {
	GetPendingProviderIDs(ctx context.Context) ([]string, error)
}

// Evaluator defines the interface for evaluating matches.
type Evaluator interface {
	Evaluate(ctx context.Context, providerID string, data []byte) error
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
	// 1. Fetch pending provider IDs from the database
	pendingIDs, err := p.repo.GetPendingProviderIDs(ctx)
	if err != nil {
		log.Printf("Failed to get pending provider IDs: %v", err)
		return
	}

	if len(pendingIDs) == 0 {
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

	// 3. Loop through pending IDs and evaluate if they are live
	for _, id := range pendingIDs {
		if data, ok := liveMatchMap[id]; ok {
			if err := p.evaluator.Evaluate(ctx, id, data); err != nil {
				log.Printf("Failed to evaluate match %s: %v", id, err)
			}
		}
	}
}
