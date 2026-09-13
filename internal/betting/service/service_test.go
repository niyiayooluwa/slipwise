package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"slipwise/internal/betting/domain"
	"slipwise/internal/betting/service"
	db "slipwise/internal/db/generated"

	"github.com/jackc/pgx/v5/pgtype"
)

type fakeRepo struct {
	globalTickets  map[uuid.UUID]*domain.SlipwiseTicket
	userTracks     map[uuid.UUID]map[uuid.UUID]bool // userID -> bookingCodeID -> tracked
	archivedTracks map[uuid.UUID]map[uuid.UUID]bool
	deletedTracks  map[uuid.UUID]map[uuid.UUID]bool
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		globalTickets:  make(map[uuid.UUID]*domain.SlipwiseTicket),
		userTracks:     make(map[uuid.UUID]map[uuid.UUID]bool),
		archivedTracks: make(map[uuid.UUID]map[uuid.UUID]bool),
		deletedTracks:  make(map[uuid.UUID]map[uuid.UUID]bool),
	}
}

func (r *fakeRepo) UpsertGlobalTicket(ctx context.Context, ticket *domain.SlipwiseTicket) (uuid.UUID, error) {
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

func (r *fakeRepo) GetUserStats(ctx context.Context, userID uuid.UUID) (db.GetUserStatsRow, error) {
	return db.GetUserStatsRow{}, nil
}

func (r *fakeRepo) GetUserHistory(ctx context.Context, arg db.GetUserHistoryParams) ([]db.GetUserHistoryRow, error) {
	var rows []db.GetUserHistoryRow
	if arg.IsArchived.Valid && arg.IsArchived.Bool {
		for id := range r.archivedTracks[arg.UserID] {
			rows = append(rows, db.GetUserHistoryRow{
				TicketID:      id,
				Provider:      "SPORTYBET",
				Code:          "J6J2TN",
				TotalOdds:     pgtype.Numeric{Valid: true},
				OverallStatus: "PENDING",
				TotalLegs:     5,
				WonLegs:       2,
				LostLegs:      1,
				PendingLegs:   2,
			})
		}
		return rows, nil
	}

	for id := range r.userTracks[arg.UserID] {
		if r.archivedTracks[arg.UserID] != nil && r.archivedTracks[arg.UserID][id] {
			continue
		}
		if r.deletedTracks[arg.UserID] != nil && r.deletedTracks[arg.UserID][id] {
			continue
		}
		rows = append(rows, db.GetUserHistoryRow{
			TicketID:      id,
			Provider:      "SPORTYBET",
			Code:          "J6J2TN",
			TotalOdds:     pgtype.Numeric{Valid: true},
			OverallStatus: "PENDING",
			TotalLegs:     5,
			WonLegs:       2,
			LostLegs:      1,
			PendingLegs:   2,
		})
	}
	return rows, nil
}

func (r *fakeRepo) CountUserHistory(ctx context.Context, arg db.CountUserHistoryParams) (int64, error) {
	if arg.IsArchived.Valid && arg.IsArchived.Bool {
		return int64(len(r.archivedTracks[arg.UserID])), nil
	}
	count := int64(0)
	for id := range r.userTracks[arg.UserID] {
		if r.archivedTracks[arg.UserID] != nil && r.archivedTracks[arg.UserID][id] {
			continue
		}
		if r.deletedTracks[arg.UserID] != nil && r.deletedTracks[arg.UserID][id] {
			continue
		}
		count++
	}
	return count, nil
}

func (r *fakeRepo) GetTicketDetails(ctx context.Context, arg db.GetTicketDetailsParams) ([]db.GetTicketDetailsRow, error) {
	return nil, nil
}

func (r *fakeRepo) DeleteUserTicket(ctx context.Context, arg db.DeleteUserTicketParams) error {
	delete(r.userTracks[arg.UserID], arg.ID)
	return nil
}

func (r *fakeRepo) BulkArchiveUserTickets(ctx context.Context, userID uuid.UUID, ticketIDs []uuid.UUID) (int64, error) {
	var count int64
	for _, id := range ticketIDs {
		if r.userTracks[userID] != nil && r.userTracks[userID][id] {
			if r.archivedTracks[userID] == nil {
				r.archivedTracks[userID] = make(map[uuid.UUID]bool)
			}
			r.archivedTracks[userID][id] = true
			count++
		}
	}
	return count, nil
}

func (r *fakeRepo) BulkUnarchiveUserTickets(ctx context.Context, userID uuid.UUID, ticketIDs []uuid.UUID) (int64, error) {
	var count int64
	for _, id := range ticketIDs {
		if r.archivedTracks[userID] != nil && r.archivedTracks[userID][id] {
			delete(r.archivedTracks[userID], id)
			count++
		}
	}
	return count, nil
}

func (r *fakeRepo) BulkSoftDeleteUserTickets(ctx context.Context, userID uuid.UUID, ticketIDs []uuid.UUID) (int64, error) {
	var count int64
	for _, id := range ticketIDs {
		if r.userTracks[userID] != nil && r.userTracks[userID][id] {
			delete(r.userTracks[userID], id)
			if r.deletedTracks[userID] == nil {
				r.deletedTracks[userID] = make(map[uuid.UUID]bool)
			}
			r.deletedTracks[userID][id] = true
			count++
		}
	}
	return count, nil
}

func (r *fakeRepo) UpdateUserTicket(ctx context.Context, arg db.UpdateUserTicketParams) (db.UserTicket, error) {
	return db.UserTicket{}, nil
}

func (r *fakeRepo) CleanupOrphanedBookingCodes(ctx context.Context) error {
	return nil
}

type fakeProvider struct{}

func (p *fakeProvider) FetchAndParse(ctx context.Context, shareCode string) (*domain.SlipwiseTicket, error) {
	return &domain.SlipwiseTicket{
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

	ctx := context.Background()
	history, _, err := svc.GetHistory(ctx, userID, 10, 0, "", nil, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 row, got %d", len(history))
	}
}

func TestBulkArchive_Success(t *testing.T) {
	repo := newFakeRepo()
	svc := service.NewBettingService(repo, nil)

	userID := uuid.New()
	codeID1 := uuid.New()
	codeID2 := uuid.New()
	repo.UpsertUserTrack(context.Background(), userID, codeID1, nil, "")
	repo.UpsertUserTrack(context.Background(), userID, codeID2, nil, "")

	ctx := context.Background()
	affected, err := svc.BulkArchiveTickets(ctx, userID, []string{codeID1.String()})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if affected != 1 {
		t.Errorf("expected 1 affected, got %d", affected)
	}

	// Active history should only have codeID2
	active, _, err := svc.GetHistory(ctx, userID, 10, 0, "", nil, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(active) != 1 || active[0].TicketID != codeID2 {
		t.Errorf("expected only codeID2 in active history, got %v", active)
	}

	// Archived history should have codeID1
	archived, _, err := svc.GetHistory(ctx, userID, 10, 0, "", nil, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(archived) != 1 || archived[0].TicketID != codeID1 {
		t.Errorf("expected codeID1 in archived history, got %v", archived)
	}
}

func TestBulkUnarchive_Success(t *testing.T) {
	repo := newFakeRepo()
	svc := service.NewBettingService(repo, nil)

	userID := uuid.New()
	codeID := uuid.New()
	repo.UpsertUserTrack(context.Background(), userID, codeID, nil, "")

	ctx := context.Background()
	svc.BulkArchiveTickets(ctx, userID, []string{codeID.String()})

	affected, err := svc.BulkUnarchiveTickets(ctx, userID, []string{codeID.String()})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if affected != 1 {
		t.Errorf("expected 1 affected, got %d", affected)
	}

	active, _, _ := svc.GetHistory(ctx, userID, 10, 0, "", nil, false)
	if len(active) != 1 {
		t.Errorf("expected 1 active ticket after unarchive, got %d", len(active))
	}
}

func TestBulkDelete_Success(t *testing.T) {
	repo := newFakeRepo()
	svc := service.NewBettingService(repo, nil)

	userID := uuid.New()
	codeID := uuid.New()
	repo.UpsertUserTrack(context.Background(), userID, codeID, nil, "")

	ctx := context.Background()
	affected, err := svc.BulkDeleteTickets(ctx, userID, []string{codeID.String()})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if affected != 1 {
		t.Errorf("expected 1 affected, got %d", affected)
	}

	active, _, _ := svc.GetHistory(ctx, userID, 10, 0, "", nil, false)
	if len(active) != 0 {
		t.Errorf("expected 0 active tickets, got %d", len(active))
	}
	archived, _, _ := svc.GetHistory(ctx, userID, 10, 0, "", nil, true)
	if len(archived) != 0 {
		t.Errorf("expected 0 archived tickets, got %d", len(archived))
	}
}

func TestBulkArchive_InvalidUUID(t *testing.T) {
	repo := newFakeRepo()
	svc := service.NewBettingService(repo, nil)

	userID := uuid.New()
	ctx := context.Background()
	_, err := svc.BulkArchiveTickets(ctx, userID, []string{"not-a-valid-uuid"})
	if err == nil {
		t.Errorf("expected error for invalid uuid, got nil")
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
