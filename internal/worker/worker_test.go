package worker

import (
	"context"
	"testing"

	"github.com/hibiken/asynq"
)

// MockDistributor is our fake sender for testing.
// It satisfies the TaskDistributor "Job Description" perfectly,
// because it has the DistributeTaskProcessGoal method!
type MockDistributor struct {
	CapturedPayload *GoalPayload // We store the data here so we can check it later
}

func (m *MockDistributor) DistributeTaskProcessGoal(ctx context.Context, payload *GoalPayload, opts ...asynq.Option) error {
	// Instead of sending to Redis across the internet, we just save it to our fake struct!
	m.CapturedPayload = payload
	return nil // Return a blank error card (Success!)
}

func TestDistributeTaskProcessGoal(t *testing.T) {
	// 1. Setup our fake distributor
	mockDistributor := &MockDistributor{}

	// 2. The data we want to send (Arsenal scores!)
	payload := &GoalPayload{
		MatchID: "12345",
		Score:   "1:1",
	}

	// 3. Call the function (In real life, the API handler would call this)
	err := mockDistributor.DistributeTaskProcessGoal(context.Background(), payload)

	// 4. Assertions: Did it work? (If anything fails, we call t.Fatalf to crash the test)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mockDistributor.CapturedPayload == nil {
		t.Fatal("expected payload to be captured, got nil")
	}

	// 5. Verify the data was perfectly handed over
	if mockDistributor.CapturedPayload.MatchID != "12345" {
		t.Errorf("expected MatchID '12345', got '%s'", mockDistributor.CapturedPayload.MatchID)
	}

	if mockDistributor.CapturedPayload.Score != "1:1" {
		t.Errorf("expected Score '1:1', got '%s'", mockDistributor.CapturedPayload.Score)
	}
}
