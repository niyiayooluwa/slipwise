package worker

import (
	"context"
	"log"
	"time"

	"slipwise/internal/betting/service"
)

// CleanupJob is the platform's scavenger routine.
// When users preview codes or delete tickets, underlying booking codes and matches
// might become orphaned (tracked by 0 users). Once a day, this job sweeps and purges
// orphaned data to keep database storage lean and clean.
type CleanupJob struct {
	svc *service.BettingService
}

func NewCleanupJob(svc *service.BettingService) *CleanupJob {
	return &CleanupJob{svc: svc}
}

func (c *CleanupJob) Start(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Cleanup job stopped")
			return
		case <-ticker.C:
			if err := c.svc.CleanupOrphanedTickets(ctx); err != nil {
				log.Printf("Failed to clean up orphaned tickets: %v", err)
			} else {
				log.Println("Successfully cleaned up orphaned tickets")
			}
		}
	}
}
