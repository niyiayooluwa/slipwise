package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"slipwise/internal/auth"
	"slipwise/internal/auth/service"
	db "slipwise/internal/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- fakeRepo: in-memory stand-in for repository.AuthRepository ---
//
// This is what the "designed to be testable" comment in service.go's
// package doc was actually for. No Postgres, no network — just maps.

var errNotFound = errors.New("not found")

type fakeRepo struct {
	mu            sync.Mutex
	usersByEmail  map[string]db.User
	usersByID     map[uuid.UUID]db.User
	latestOTP     map[string]db.OtpCode // key: email + "|" + purpose
	refreshTokens map[string]db.RefreshToken
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		usersByEmail:  map[string]db.User{},
		usersByID:     map[uuid.UUID]db.User{},
		latestOTP:     map[string]db.OtpCode{},
		refreshTokens: map[string]db.RefreshToken{},
	}
}

func (f *fakeRepo) CreateUser(_ context.Context, username, email string, passwordHash *string) (db.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var pgUsername pgtype.Text
	if username != "" {
		pgUsername = pgtype.Text{String: username, Valid: true}
	}

	u := db.User{ID: uuid.New(), Username: pgUsername, Email: email, PasswordHash: passwordHash, CreatedAt: time.Now()}
	f.usersByEmail[email] = u
	f.usersByID[u.ID] = u
	return u, nil
}

func (f *fakeRepo) UpdateUserProfile(_ context.Context, id uuid.UUID, username *string) (db.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.usersByID[id]
	if !ok {
		return db.User{}, errNotFound
	}
	if username != nil {
		u.Username = pgtype.Text{String: *username, Valid: true}
	}
	f.usersByID[id] = u
	f.usersByEmail[u.Email] = u
	return u, nil
}

func (f *fakeRepo) GetUserByEmail(_ context.Context, email string) (db.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.usersByEmail[email]
	if !ok {
		return db.User{}, errNotFound
	}
	return u, nil
}

func (f *fakeRepo) GetUserByID(_ context.Context, id uuid.UUID) (db.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.usersByID[id]
	if !ok {
		return db.User{}, errNotFound
	}
	return u, nil
}

func (f *fakeRepo) MarkEmailVerified(_ context.Context, userID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.usersByID[userID]
	if !ok {
		return errNotFound
	}
	now := time.Now()
	u.EmailVerifiedAt = &now
	f.usersByID[userID] = u
	f.usersByEmail[u.Email] = u
	return nil
}

func (f *fakeRepo) UpdateUserPassword(_ context.Context, email string, passwordHash *string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.usersByEmail[email]
	if !ok {
		return errNotFound
	}
	u.PasswordHash = passwordHash
	f.usersByEmail[email] = u
	f.usersByID[u.ID] = u
	return nil
}

func (f *fakeRepo) CreateOTP(_ context.Context, email, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o := db.OtpCode{
		ID:        uuid.New(),
		Email:     email,
		CodeHash:  codeHash,
		Purpose:   purpose,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	f.latestOTP[email+"|"+purpose] = o
	return o, nil
}

func (f *fakeRepo) GetLatestOTP(_ context.Context, email, purpose string) (db.OtpCode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.latestOTP[email+"|"+purpose]
	if !ok || o.UsedAt != nil {
		return db.OtpCode{}, errNotFound
	}
	return o, nil
}

func (f *fakeRepo) IncrementOTPAttempts(_ context.Context, otpID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, o := range f.latestOTP {
		if o.ID == otpID {
			o.AttemptCount++
			f.latestOTP[k] = o
			return nil
		}
	}
	return errNotFound
}

func (f *fakeRepo) MarkOTPUsed(_ context.Context, otpID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, o := range f.latestOTP {
		if o.ID == otpID {
			now := time.Now()
			o.UsedAt = &now
			f.latestOTP[k] = o
			return nil
		}
	}
	return errNotFound
}

func (f *fakeRepo) CreateRefreshToken(_ context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t := db.RefreshToken{ID: uuid.New(), UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt, CreatedAt: time.Now()}
	f.refreshTokens[tokenHash] = t
	return t, nil
}

func (f *fakeRepo) GetRefreshToken(_ context.Context, tokenHash string) (db.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.refreshTokens[tokenHash]
	if !ok {
		return db.RefreshToken{}, errNotFound
	}
	return t, nil
}

func (f *fakeRepo) RevokeRefreshToken(_ context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, t := range f.refreshTokens {
		if t.ID == id {
			now := time.Now()
			t.RevokedAt = &now
			f.refreshTokens[k] = t
			return nil
		}
	}
	return errNotFound
}

func (f *fakeRepo) RevokeAllUserRefreshTokens(_ context.Context, userID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, t := range f.refreshTokens {
		if t.UserID == userID && t.RevokedAt == nil {
			now := time.Now()
			t.RevokedAt = &now
			f.refreshTokens[k] = t
		}
	}
	return nil
}

func (f *fakeRepo) CreateOAuthConnection(_ context.Context, userID uuid.UUID, provider, providerUserID string) error {
	return nil
}

func (f *fakeRepo) GetUserByOAuthProvider(_ context.Context, provider, providerUserID string) (db.User, error) {
	return db.User{}, errNotFound
}

func (f *fakeRepo) GetOAuthProvidersForUser(_ context.Context, userID uuid.UUID) ([]string, error) {
	return nil, nil
}

// --- fakeMailer: captures sends instead of calling Resend ---

type fakeMailer struct {
	mu   sync.Mutex
	sent []sentOTP
}

type sentOTP struct {
	email, code string
}

func (m *fakeMailer) SendOTP(_ context.Context, email, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, sentOTP{email: email, code: code})
	return nil
}

func (m *fakeMailer) lastCode(t *testing.T) string {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) == 0 {
		t.Fatal("expected an OTP to have been sent, none were")
	}
	return m.sent[len(m.sent)-1].code
}

// --- fake repo CheckUsernameExists ---
func (f *fakeRepo) CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if username == "taken" {
		return true, nil
	}
	return false, nil
}

// --- test setup helper ---

func newTestService() (*service.AuthService, *fakeRepo, *fakeMailer) {
	repo := newFakeRepo()
	mailer := &fakeMailer{}
	issuer := auth.NewJWTIssuer("test-secret")
	googleClientID := "test-string"
	return service.NewAuthService(repo, issuer, mailer, googleClientID, "test@admin.com"), repo, mailer
}

// --- Signup ---

func TestSignup_Success(t *testing.T) {
	svc, repo, mailer := newTestService()
	ctx := context.Background()

	err := svc.Signup(ctx, "johndoe", "new@example.com", "password123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := repo.usersByEmail["new@example.com"]; !ok {
		t.Fatal("expected user row to exist after signup")
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("expected exactly 1 OTP sent, got %d", len(mailer.sent))
	}
}

func TestSignup_DuplicateEmail(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "dupe@example.com", "password123")
	err := svc.Signup(ctx, "johndoe", "dupe@example.com", "password123")

	if !errors.Is(err, service.ErrEmailAlreadyRegistered) {
		t.Fatalf("expected ErrEmailAlreadyRegistered, got %v", err)
	}
}

// --- Verify ---

func TestVerify_Success(t *testing.T) {
	svc, _, mailer := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "verify@example.com", "password123")
	code := mailer.lastCode(t)

	pair, err := svc.Verify(ctx, "verify@example.com", code)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected both tokens to be populated")
	}
}

func TestVerify_WrongCode(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "wrongcode@example.com", "password123")

	_, err := svc.Verify(ctx, "wrongcode@example.com", "000000")
	if !errors.Is(err, service.ErrOTPIncorrect) {
		t.Fatalf("expected ErrOTPIncorrect, got %v", err)
	}
}

func TestVerify_MaxAttemptsLocksCode(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "maxattempts@example.com", "password123")

	// 3 wrong guesses burns the code...
	for i := 0; i < 3; i++ {
		_, err := svc.Verify(ctx, "maxattempts@example.com", "000000")
		if !errors.Is(err, service.ErrOTPIncorrect) {
			t.Fatalf("attempt %d: expected ErrOTPIncorrect, got %v", i+1, err)
		}
	}

	// ...the 4th attempt should be locked out, not "incorrect" again,
	// even against a guess that would otherwise be irrelevant.
	_, err := svc.Verify(ctx, "maxattempts@example.com", "111111")
	if !errors.Is(err, service.ErrOTPMaxAttempts) {
		t.Fatalf("expected ErrOTPMaxAttempts after 3 failures, got %v", err)
	}
}

func TestVerify_ExpiredCode(t *testing.T) {
	svc, repo, mailer := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "expired@example.com", "password123")
	code := mailer.lastCode(t)

	// Reach into the fake to simulate time having passed, rather than
	// sleeping 10 real minutes in a test.
	repo.mu.Lock()
	otp := repo.latestOTP["expired@example.com|signup_verify"]
	otp.ExpiresAt = time.Now().Add(-1 * time.Minute)
	repo.latestOTP["expired@example.com|signup_verify"] = otp
	repo.mu.Unlock()

	_, err := svc.Verify(ctx, "expired@example.com", code)
	if !errors.Is(err, service.ErrOTPExpired) {
		t.Fatalf("expected ErrOTPExpired, got %v", err)
	}
}

// --- Login ---

func TestLogin_Success(t *testing.T) {
	svc, _, mailer := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "login@example.com", "password123")
	code := mailer.lastCode(t)
	_, _ = svc.Verify(ctx, "login@example.com", code)

	pair, err := svc.Login(ctx, "login@example.com", "password123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pair.AccessToken == "" {
		t.Fatal("expected an access token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, _, mailer := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "wrongpw@example.com", "password123")
	code := mailer.lastCode(t)
	_, _ = svc.Verify(ctx, "wrongpw@example.com", code)

	_, err := svc.Login(ctx, "wrongpw@example.com", "not-the-password")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownEmail_SameErrorAsWrongPassword(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	// This is the anti-enumeration guarantee documented in errors.go —
	// asserting it here means a future refactor that breaks it fails
	// a test, not just a code review.
	_, err := svc.Login(ctx, "never-signed-up@example.com", "whatever123")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for unknown email, got %v", err)
	}
}

func TestLogin_UnverifiedAccount(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "unverified@example.com", "password123")
	// deliberately never call Verify

	_, err := svc.Login(ctx, "unverified@example.com", "password123")
	if !errors.Is(err, service.ErrEmailNotVerified) {
		t.Fatalf("expected ErrEmailNotVerified, got %v", err)
	}
}

// --- Refresh ---

func TestRefresh_RotatesToken(t *testing.T) {
	svc, _, mailer := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "refresh@example.com", "password123")
	code := mailer.lastCode(t)
	firstPair, _ := svc.Verify(ctx, "refresh@example.com", code)

	secondPair, err := svc.Refresh(ctx, firstPair.RefreshToken)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if secondPair.RefreshToken == firstPair.RefreshToken {
		t.Fatal("expected a new refresh token, got the same one back")
	}

	// The old token should now be dead — this is the rotation
	// guarantee, not just "a new one also works."
	_, err = svc.Refresh(ctx, firstPair.RefreshToken)
	if !errors.Is(err, service.ErrRefreshTokenInvalid) {
		t.Fatalf("expected old refresh token to be invalid after rotation, got %v", err)
	}
}

func TestRefresh_ReuseTriggersGlobalNuke(t *testing.T) {
	svc, _, mailer := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "nuke@example.com", "password123")
	code := mailer.lastCode(t)

	// Legitimate login 1 (Phone A)
	pair1, _ := svc.Verify(ctx, "nuke@example.com", code)
	// Legitimate login 2 (Phone B)
	pair2, _ := svc.Login(ctx, "nuke@example.com", "password123")

	// Phone A rotates token normally
	pair1Rotated, err := svc.Refresh(ctx, pair1.RefreshToken)
	if err != nil {
		t.Fatalf("expected no error on normal refresh, got %v", err)
	}

	// Hacker steals the old, already-used token from Phone A and tries to use it
	_, err = svc.Refresh(ctx, pair1.RefreshToken)
	if !errors.Is(err, service.ErrRefreshTokenInvalid) {
		t.Fatalf("expected ErrRefreshTokenInvalid on reuse, got %v", err)
	}

	// The trap sprang! Both the rotated Phone A token AND Phone B's token should now be dead
	_, err = svc.Refresh(ctx, pair1Rotated.RefreshToken)
	if !errors.Is(err, service.ErrRefreshTokenInvalid) {
		t.Fatal("expected Phone A rotated token to be nuked, but it was still valid")
	}

	_, err = svc.Refresh(ctx, pair2.RefreshToken)
	if !errors.Is(err, service.ErrRefreshTokenInvalid) {
		t.Fatal("expected Phone B token to be nuked, but it was still valid")
	}
}

func TestRefresh_UnknownToken(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Refresh(ctx, "not-a-real-token")
	if !errors.Is(err, service.ErrRefreshTokenInvalid) {
		t.Fatalf("expected ErrRefreshTokenInvalid, got %v", err)
	}
}

// --- Logout ---

func TestLogout_IsIdempotent(t *testing.T) {
	svc, _, mailer := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "logout@example.com", "password123")
	code := mailer.lastCode(t)
	pair, _ := svc.Verify(ctx, "logout@example.com", code)

	if err := svc.Logout(ctx, pair.RefreshToken); err != nil {
		t.Fatalf("expected no error on first logout, got %v", err)
	}
	// Logging out again with the same (now-revoked) token should still
	// succeed silently — that's the documented contract.
	if err := svc.Logout(ctx, pair.RefreshToken); err != nil {
		t.Fatalf("expected no error on second logout, got %v", err)
	}
	// Logging out with a token that never existed should also succeed.
	if err := svc.Logout(ctx, "never-existed"); err != nil {
		t.Fatalf("expected no error for unknown token, got %v", err)
	}
}

// --- ResendOTP ---

func TestResendOTP_Success(t *testing.T) {
	svc, repo, mailer := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "resend@example.com", "password123")

	// Simulate the cooldown window having already passed.
	repo.mu.Lock()
	otp := repo.latestOTP["resend@example.com|signup_verify"]
	otp.CreatedAt = time.Now().Add(-2 * time.Minute)
	repo.latestOTP["resend@example.com|signup_verify"] = otp
	repo.mu.Unlock()

	err := svc.ResendOTP(ctx, "resend@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(mailer.sent) != 2 {
		t.Fatalf("expected 2 total sends (signup + resend), got %d", len(mailer.sent))
	}
}

func TestResendOTP_Cooldown(t *testing.T) {
	svc, _, mailer := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "cooldown@example.com", "password123")
	// No time manipulation — the OTP from Signup was just created, so
	// this should still be inside the 60s cooldown window.

	err := svc.ResendOTP(ctx, "cooldown@example.com")
	if !errors.Is(err, service.ErrOTPCooldown) {
		t.Fatalf("expected ErrOTPCooldown, got %v", err)
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("expected no additional send during cooldown, got %d total", len(mailer.sent))
	}
}

func TestResendOTP_UnknownEmail(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	err := svc.ResendOTP(ctx, "never-signed-up@example.com")
	if !errors.Is(err, service.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestResendOTP_AlreadyVerified(t *testing.T) {
	svc, _, mailer := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "alreadyverified@example.com", "password123")
	code := mailer.lastCode(t)
	_, _ = svc.Verify(ctx, "alreadyverified@example.com", code)

	err := svc.ResendOTP(ctx, "alreadyverified@example.com")
	if !errors.Is(err, service.ErrAlreadyVerified) {
		t.Fatalf("expected ErrAlreadyVerified, got %v", err)
	}
}

// --- ForgotPassword ---

func TestForgotPassword_Success(t *testing.T) {
	svc, repo, mailer := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "forgot@example.com", "password123")

	// Reset mailer since Signup sends an OTP
	mailer.mu.Lock()
	mailer.sent = nil
	mailer.mu.Unlock()

	err := svc.ForgotPassword(ctx, "forgot@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("expected 1 OTP sent, got %d", len(mailer.sent))
	}

	// Verify purpose in DB
	repo.mu.Lock()
	otp, ok := repo.latestOTP["forgot@example.com|password_reset"]
	repo.mu.Unlock()
	if !ok {
		t.Fatal("expected password_reset OTP in DB")
	}
	if otp.Purpose != "password_reset" {
		t.Fatalf("expected purpose password_reset, got %s", otp.Purpose)
	}
}

func TestForgotPassword_UnknownEmail_SilentlySucceeds(t *testing.T) {
	svc, _, mailer := newTestService()
	ctx := context.Background()

	err := svc.ForgotPassword(ctx, "unknown@example.com")
	if err != nil {
		t.Fatalf("expected no error to prevent enumeration, got %v", err)
	}
	if len(mailer.sent) != 0 {
		t.Fatalf("expected 0 OTP sent, got %d", len(mailer.sent))
	}
}

// --- ResetPassword ---

func TestResetPassword_Success(t *testing.T) {
	svc, _, mailer := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "reset@example.com", "password123")
	_ = svc.ForgotPassword(ctx, "reset@example.com")
	code := mailer.lastCode(t)

	err := svc.ResetPassword(ctx, "reset@example.com", code, "newpassword456")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify new password works
	_, err = svc.Login(ctx, "reset@example.com", "newpassword456")
	// Since we didn't verify the email in this test before login, it will return ErrEmailNotVerified
	// But it won't return ErrInvalidCredentials, which means password is correct.
	if errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatal("expected new password to work, but got ErrInvalidCredentials")
	}
}

func TestResetPassword_WrongCode(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	_ = svc.Signup(ctx, "johndoe", "resetwrong@example.com", "password123")
	_ = svc.ForgotPassword(ctx, "resetwrong@example.com")

	err := svc.ResetPassword(ctx, "resetwrong@example.com", "000000", "newpassword")
	if !errors.Is(err, service.ErrOTPIncorrect) {
		t.Fatalf("expected ErrOTPIncorrect, got %v", err)
	}
}

// --- CheckUsername ---

func TestCheckUsername_Available(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	available, err := svc.CheckUsername(ctx, "new_user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !available {
		t.Fatal("expected username to be available")
	}
}

func TestCheckUsername_Taken(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	available, err := svc.CheckUsername(ctx, "taken")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if available {
		t.Fatal("expected 'taken' username to be unavailable")
	}
}

func TestCheckUsername_Empty(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	available, err := svc.CheckUsername(ctx, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if available {
		t.Fatal("expected empty username to be unavailable")
	}
}

func (m *fakeMailer) SendFeedback(ctx context.Context, toEmail, userEmail, feedback string) error {
	return nil
}
