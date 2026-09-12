package worker

import (
	"context"
	"testing"

	db "slipwise/internal/db/generated"

	"github.com/google/uuid"
)

type mockEvaluatorRepo struct {
	buckets            []db.GetPendingBucketsForMatchRow
	updatedSelections  []db.UpdateSelectionStatusParams
	returnedTickets    []uuid.UUID
	evalTicketsArg     []uuid.UUID
	evalTicketsResults []db.EvaluateTicketsRow
}

func (m *mockEvaluatorRepo) GetPendingBucketsForMatch(ctx context.Context, matchID uuid.UUID) ([]db.GetPendingBucketsForMatchRow, error) {
	return m.buckets, nil
}

func (m *mockEvaluatorRepo) UpdateSelectionStatus(ctx context.Context, arg db.UpdateSelectionStatusParams) ([]uuid.UUID, error) {
	m.updatedSelections = append(m.updatedSelections, arg)
	return m.returnedTickets, nil
}

func (m *mockEvaluatorRepo) EvaluateTickets(ctx context.Context, bookingCodeIds []uuid.UUID) ([]db.EvaluateTicketsRow, error) {
	m.evalTicketsArg = bookingCodeIds
	return m.evalTicketsResults, nil
}

func (m *mockEvaluatorRepo) UpdateMatchState(ctx context.Context, arg db.UpdateMatchStateParams) error {
	return nil
}

func (m *mockEvaluatorRepo) GetMatchByID(ctx context.Context, id uuid.UUID) (db.GetMatchByIDRow, error) {
	return db.GetMatchByIDRow{}, nil
}

func (m *mockEvaluatorRepo) GetPendingSelectionsForMatch(ctx context.Context, matchID uuid.UUID) ([]db.GetPendingSelectionsForMatchRow, error) {
	return nil, nil
}

func (m *mockEvaluatorRepo) SetEarlyWinNotified(ctx context.Context, arg db.SetEarlyWinNotifiedParams) error {
	return nil
}

func (m *mockEvaluatorRepo) SetHTNotified(ctx context.Context, arg db.SetHTNotifiedParams) error {
	return nil
}

type mockNotificationService struct {
	sentTokens []string
	sentTitles []string
}

func (m *mockNotificationService) SendMulticast(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	m.sentTokens = append(m.sentTokens, tokens...)
	m.sentTitles = append(m.sentTitles, title)
	return nil
}

func TestLiveEvaluator_NotificationScenarios(t *testing.T) {
	matchID := uuid.New()
	providerID := "sr:match:123"

	// Create payload (Home team won 2-1)
	payload := []byte(`{"setScore":"2:1","matchStatus":"ENDED"}`)

	token := "token123"

	tests := []struct {
		name               string
		stats              db.EvaluateTicketsRow
		expectNotification bool
	}{
		{
			name: "Ticket Won",
			stats: db.EvaluateTicketsRow{
				TicketStatus:  "WON",
				TotalLegs:     10,
				PendingLegs:   0,
				LostLegs:      0,
				BookingCodeID: uuid.New(),
				UserTicketID:  uuid.New(),
				FcmToken:      &token,
			},
			expectNotification: true,
		},
		{
			name: "Ticket Lost",
			stats: db.EvaluateTicketsRow{
				TicketStatus:  "LOST",
				TotalLegs:     10,
				LostLegs:      1,
				BookingCodeID: uuid.New(),
				UserTicketID:  uuid.New(),
				FcmToken:      &token,
			},
			expectNotification: true,
		},
		{
			name: "Pending Ticket (No notification)",
			stats: db.EvaluateTicketsRow{
				TicketStatus:  "PENDING",
				TotalLegs:     10,
				PendingLegs:   1,
				BookingCodeID: uuid.New(),
				UserTicketID:  uuid.New(),
				FcmToken:      &token,
			},
			expectNotification: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockEvaluatorRepo{
				buckets: []db.GetPendingBucketsForMatchRow{
					{MarketType: "MATCH_RESULT", Selection: "1"},
				},
				returnedTickets:    []uuid.UUID{tt.stats.BookingCodeID},
				evalTicketsResults: []db.EvaluateTicketsRow{tt.stats},
			}
			fcm := &mockNotificationService{}

			evaluator := NewLiveEvaluator(repo, fcm)

			err := evaluator.Evaluate(context.Background(), matchID, providerID, payload)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.expectNotification {
				if len(fcm.sentTitles) == 0 {
					t.Fatalf("Expected a notification to be sent, but none was sent")
				}
			} else {
				if len(fcm.sentTitles) != 0 {
					t.Fatalf("Expected NO notification, but got %d", len(fcm.sentTitles))
				}
			}
		})
	}
}
