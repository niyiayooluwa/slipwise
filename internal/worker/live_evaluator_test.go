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
		name          string
		stats         db.EvaluateTicketsRow
		expectedTitle string
	}{
		{
			name: "Ticket Won",
			stats: db.EvaluateTicketsRow{
				TicketStatus:  "WON",
				TotalLegs:     10,
				PendingLegs:   0,
				LostLegs:      0,
				BookingCodeID: uuid.New(),
				FcmToken:      &token,
			},
			expectedTitle: "Ticket Won! \U0001f4b8\U0001f680",
		},
		{
			name: "Ticket Lost",
			stats: db.EvaluateTicketsRow{
				TicketStatus:  "LOST",
				TotalLegs:     10,
				LostLegs:      1,
				BookingCodeID: uuid.New(),
				FcmToken:      &token,
			},
			expectedTitle: "Ticket Lost \u274c",
		},

	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockEvaluatorRepo{
				buckets: []db.GetPendingBucketsForMatchRow{
					{MarketType: "MATCH_RESULT", Selection: "1"}, // Matches 2:1 home win
				},
				returnedTickets:    []uuid.UUID{tt.stats.BookingCodeID},
				evalTicketsResults: []db.EvaluateTicketsRow{tt.stats},
			}
			fcm := &mockNotificationService{}
			evaluator := NewLiveEvaluator(repo, fcm)

			err := evaluator.Evaluate(context.Background(), matchID, providerID, payload)
			if err != nil {
				t.Fatalf("Evaluate returned error: %v", err)
			}

			if len(fcm.sentTitles) == 0 {
				t.Fatalf("expected notification, got none")
			}

			if fcm.sentTitles[0] != tt.expectedTitle {
				t.Errorf("expected title '%s', got '%s'", tt.expectedTitle, fcm.sentTitles[0])
			}
			if fcm.sentTokens[0] != "token123" {
				t.Errorf("expected token 'token123', got '%s'", fcm.sentTokens[0])
			}
		})
	}
}
