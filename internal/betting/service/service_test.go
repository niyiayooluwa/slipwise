package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"sportloga/internal/betting/domain"
	"sportloga/internal/betting/service"
	db "sportloga/internal/db/generated"

	"github.com/jackc/pgx/v5/pgtype"
)

type fakeRepo struct {
	globalTickets map[uuid.UUID]*domain.SportlogaTicket
	userTracks    map[uuid.UUID]map[uuid.UUID]bool // userID -> bookingCodeID -> tracked
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		globalTickets: make(map[uuid.UUID]*domain.SportlogaTicket),
		userTracks:    make(map[uuid.UUID]map[uuid.UUID]bool),
	}
}

func (r *fakeRepo) UpsertGlobalTicket(ctx context.Context, ticket *domain.SportlogaTicket) (uuid.UUID, error) {
	id := uuid.New()
	r.globalTickets[id] = ticket
	return id, nil
}

func (r *fakeRepo) UpsertUserTrack(ctx context.Context, userID, bookingCodeID uuid.UUID, stake *float64, description string) error {
	if _, ok := r.userTracks[userID]; !ok {
		r.userTracks[userID] = make(map[uuid.UUID]bool)
	}
	r.userTracks[userID][bookingCodeID] = true
	return nil
}

func (r *fakeRepo) GetUserHistory(ctx context.Context, userID uuid.UUID) ([]db.GetUserHistoryRow, error) {
	var rows []db.GetUserHistoryRow
	for id := range r.userTracks[userID] {
		rows = append(rows, db.GetUserHistoryRow{
			TicketID:      id,
			Provider:      "SPORTYBET",
			Code:          "J6J2TN",
			TotalOdds:     pgtype.Numeric{Valid: true},
			OverallStatus: "PENDING",
		})
	}
	return rows, nil
}

func (r *fakeRepo) GetTicketDetails(ctx context.Context, arg db.GetTicketDetailsParams) ([]db.GetTicketDetailsRow, error) {
	return nil, nil
}

func (r *fakeRepo) DeleteUserTicket(ctx context.Context, arg db.DeleteUserTicketParams) error {
	delete(r.userTracks[arg.UserID], arg.ID)
	return nil
}

func (r *fakeRepo) CleanupOrphanedBookingCodes(ctx context.Context) error {
	return nil
}

type fakeProvider struct{}

func (p *fakeProvider) FetchAndParse(ctx context.Context, shareCode string) (*domain.SportlogaTicket, error) {
	return &domain.SportlogaTicket{
		Provider:  "SPORTYBET",
		Code:      shareCode,
		TotalOdds: 1.5,
	}, nil
}

func TestPreviewTicket_Success(t *testing.T) {
	repo := newFakeRepo()
	providers := map[string]service.TicketProvider{
		"SPORTYBET": &fakeProvider{},
	}
	svc := service.NewBettingService(repo, providers)

	res, err := svc.PreviewTicket(context.Background(), "SPORTYBET", "J6J2TN")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.BookingCodeID == uuid.Nil {
		t.Errorf("expected non-nil uuid")
	}
	if res.Ticket.Code != "J6J2TN" {
		t.Errorf("expected code J6J2TN, got %s", res.Ticket.Code)
	}
}

func TestPreviewTicket_UnsupportedProvider(t *testing.T) {
	repo := newFakeRepo()
	providers := map[string]service.TicketProvider{}
	svc := service.NewBettingService(repo, providers)

	_, err := svc.PreviewTicket(context.Background(), "UNKNOWN", "123")
	if err != service.ErrUnsupportedProvider {
		t.Fatalf("expected ErrUnsupportedProvider, got %v", err)
	}
}

func TestTrackTicket_Success(t *testing.T) {
	repo := newFakeRepo()
	svc := service.NewBettingService(repo, nil)

	userID := uuid.New()
	codeID := uuid.New()
	err := svc.TrackTicket(context.Background(), userID, codeID, nil, "Test")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.userTracks[userID][codeID] {
		t.Errorf("expected ticket to be tracked")
	}
}

func TestGetHistory_ReturnsRows(t *testing.T) {
	repo := newFakeRepo()
	svc := service.NewBettingService(repo, nil)

	userID := uuid.New()
	codeID := uuid.New()
	repo.UpsertUserTrack(context.Background(), userID, codeID, nil, "")

	rows, err := svc.GetHistory(context.Background(), userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}
}

func TestDeleteTicket_RemovesLink(t *testing.T) {
	repo := newFakeRepo()
	svc := service.NewBettingService(repo, nil)

	userID := uuid.New()
	codeID := uuid.New()
	repo.UpsertUserTrack(context.Background(), userID, codeID, nil, "")

	err := svc.DeleteTicket(context.Background(), db.DeleteUserTicketParams{
		ID:     codeID,
		UserID: userID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.userTracks[userID][codeID] {
		t.Errorf("expected ticket to be deleted")
	}
}
