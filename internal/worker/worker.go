package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

const (
	TaskProcessGoal = "task:process_goal"
)

// GoalPayload is the data we pass to the background job
type GoalPayload struct {
	MatchID string `json:"match_id"`
	Score   string `json:"score"`
}

// -----------------------------------------------------------------------------
// The Distributor (The Sender)
// -----------------------------------------------------------------------------

// TaskDistributor is the "Job Description" (Interface) for sending tasks.
// Because we use an interface, we can fake this in our tests.
type TaskDistributor interface {
	DistributeTaskProcessGoal(ctx context.Context, payload *GoalPayload, opts ...asynq.Option) error
}

// RedisTaskDistributor is the actual worker that talks to Redis in production.
type RedisTaskDistributor struct {
	client *asynq.Client
}

func NewRedisTaskDistributor(redisOpt asynq.RedisClientOpt) TaskDistributor {
	client := asynq.NewClient(redisOpt)
	return &RedisTaskDistributor{
		client: client,
	}
}

func (d *RedisTaskDistributor) DistributeTaskProcessGoal(ctx context.Context, payload *GoalPayload, opts ...asynq.Option) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task payload: %w", err) // The error card!
	}

	task := asynq.NewTask(TaskProcessGoal, jsonPayload, opts...)

	// Talk to the real Redis here
	info, err := d.client.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err) // Another error card!
	}

	slog.Info("enqueued task", "type", task.Type(), "queue", info.Queue)
	return nil // Success, return a blank error card
}
