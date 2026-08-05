// Package service holds the auth domain's business logic. It knows
// about OTP expiry rules, token rotation, and verification gating —
// but nothing about HTTP (that's handler) and nothing about SQL
// (that's repository, consumed here only through the AuthRepository
// interface). This is the layer to unit test: swap in a fake
// AuthRepository and a fake Mailer, no network or DB required.
package service

import (
	"context"
	"time"

	"sportloga/internal/auth"
	"sportloga/internal/auth/repository"

	"github.com/google/uuid"
	"google.golang.org/api/idtoken"

	db "sportloga/internal/db/generated"
)

// otpTTL is how long a signup OTP stays valid after issuance.
const otpTTL = 10 * time.Minute

// otpMaxAttempts is how many wrong guesses are allowed against a
// single OTP before it's dead and a new one must be requested.
const otpMaxAttempts = 3

// otpResendCooldown is the minimum time between OTP sends for the same
// email — stops a caller (or a user mashing "resend") from hammering
// the mail provider and the recipient's inbox.
const otpResendCooldown = 60 * time.Second

// otpPurposeSignup tags OTPs issued for signup email verification,
// distinct from any future purpose (e.g. password reset) that might
// share the same otp_codes table.
const otpPurposeSignup = "signup_verify"

// refreshTokenTTLDays is how long a refresh token stays valid after
// issuance, and after every rotation.
const refreshTokenTTLDays = 30

// Mailer sends the OTP email. Implemented by an internal/mailer
// Resend wrapper; kept as an interface here so the service can be
// tested without sending real email.
type Mailer interface {
	// SendOTP delivers code to email. Errors here should be treated as
	// fatal to the calling request — an OTP that isn't actually
	// delivered is a broken signup, not a soft failure.
	SendOTP(ctx context.Context, email, code string) error
}

// TokenPair is what every successful auth operation (verify, login,
// refresh) hands back. It is service's own type — deliberately not
// model.TokenPairResponse — so this package has no dependency on the
// HTTP-layer DTOs.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// UserProfile is the service-layer user shape returned to the handler.
// It deliberately does not expose the password hash or internal DB types.
type UserProfile struct {
	ID         uuid.UUID
	FirstName  string
	LastName   string
	Email      string
	IsVerified bool
}

// AuthService implements signup/verify/login/refresh/logout. Construct
// with NewAuthService; all fields are unexported to keep the
// dependency list explicit at the constructor rather than settable ad
// hoc.
type AuthService struct {
	repo           repository.AuthRepository
	issuer         *auth.JWTIssuer
	mailer         Mailer
	googleClientID string
}

// NewAuthService wires an AuthService from its dependencies.
func NewAuthService(repo repository.AuthRepository, issuer *auth.JWTIssuer, mailer Mailer, googleClientID string) *AuthService {
	return &AuthService{repo: repo, issuer: issuer, mailer: mailer, googleClientID: googleClientID}
}

// Signup creates a new, unverified user and sends a signup OTP to
// email. The user row is created immediately (with
// email_verified_at = null) rather than deferred until Verify
// succeeds — this trades "a signup that's abandoned mid-OTP leaves a
// dangling unverified row" for "an OTP always has a real user_id to
// belong to." Returns ErrEmailAlreadyRegistered if email is taken.
func (s *AuthService) Signup(ctx context.Context, firstName, lastName, email, password string) error {
	if _, err := s.repo.GetUserByEmail(ctx, email); err == nil {
		return ErrEmailAlreadyRegistered
	}

	pwHash, err := auth.HashSecret(password)
	if err != nil {
		return err
	}

	user, err := s.repo.CreateUser(ctx, firstName, lastName, email, &pwHash)
	if err != nil {
		return err
	}

	return s.issueAndSendOTP(ctx, user.Email)
}

// issueAndSendOTP generates a code, stores its bcrypt hash, and emails
// the raw code. The raw code never touches the database — only
// CheckSecret against the stored hash can confirm a match.
func (s *AuthService) issueAndSendOTP(ctx context.Context, email string) error {
	code, err := auth.GenerateOTP()
	if err != nil {
		return err
	}
	codeHash, err := auth.HashSecret(code)
	if err != nil {
		return err
	}
	if _, err := s.repo.CreateOTP(ctx, email, codeHash, otpPurposeSignup, time.Now().Add(otpTTL)); err != nil {
		return err
	}
	return s.mailer.SendOTP(ctx, email, code)
}

// Verify checks a signup OTP and, on success, marks the account
// verified and issues the first token pair. Wrong-guess attempts are
// counted even on failure so otpMaxAttempts is enforced across calls,
// not just within one.
func (s *AuthService) Verify(ctx context.Context, email, code string) (TokenPair, error) {
	otp, err := s.repo.GetLatestOTP(ctx, email, otpPurposeSignup)
	if err != nil {
		return TokenPair{}, ErrOTPNotFound
	}
	if time.Now().After(otp.ExpiresAt) {
		return TokenPair{}, ErrOTPExpired
	}
	if int(otp.AttemptCount) >= otpMaxAttempts {
		return TokenPair{}, ErrOTPMaxAttempts
	}
	if !auth.CheckSecret(otp.CodeHash, code) {
		_ = s.repo.IncrementOTPAttempts(ctx, otp.ID)
		return TokenPair{}, ErrOTPIncorrect
	}

	_ = s.repo.MarkOTPUsed(ctx, otp.ID)

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return TokenPair{}, err
	}
	if err := s.repo.MarkEmailVerified(ctx, user.ID); err != nil {
		return TokenPair{}, err
	}

	return s.issueTokenPair(ctx, user.ID)
}

// ResendOTP issues a fresh signup OTP for an account that never
// completed verification — the self-service escape hatch for a code
// that expired or got burned through 3 wrong guesses. Deliberately
// does NOT touch/revoke the previous OTP row; GetLatestOTP always
// picks the most recently created unused one, so the old code simply
// stops being reachable once a newer one exists.
func (s *AuthService) ResendOTP(ctx context.Context, email string) error {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return ErrUserNotFound
	}
	if user.EmailVerifiedAt != nil {
		return ErrAlreadyVerified
	}

	latest, err := s.repo.GetLatestOTP(ctx, email, otpPurposeSignup)
	if err == nil && time.Since(latest.CreatedAt) < otpResendCooldown {
		return ErrOTPCooldown
	}
	// err != nil here just means there's no live (unused) OTP yet —
	// that's the normal case, not a failure, so we fall through and
	// issue one.

	return s.issueAndSendOTP(ctx, email)
}

// Login checks password and verification status, then issues a fresh
// token pair. Returns ErrInvalidCredentials for both "no such user"
// and "wrong password" (see errors.go for why), and
// ErrEmailNotVerified separately since that's not a secret — the
// account's existence is already implied by the correct password.
func (s *AuthService) Login(ctx context.Context, email, password string) (TokenPair, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)

	// If they exist but have no password, they likely signed up via OAuth.
	if err == nil && user.PasswordHash == nil {
		providers, pErr := s.repo.GetOAuthProvidersForUser(ctx, user.ID)
		if pErr == nil && len(providers) > 0 {
			return TokenPair{}, ErrOAuthAccount
		}
	}

	if err != nil || user.PasswordHash == nil || !auth.CheckSecret(*user.PasswordHash, password) {
		return TokenPair{}, ErrInvalidCredentials
	}
	if user.EmailVerifiedAt == nil {
		return TokenPair{}, ErrEmailNotVerified
	}
	return s.issueTokenPair(ctx, user.ID)
}

// Refresh rotates a refresh token: the presented one is revoked and a
// brand new pair is issued. Rotation (rather than reuse) means a
// stolen refresh token only has a window until the legitimate user's
// next refresh call, at which point it stops working.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	hash := auth.HashToken(refreshToken)
	stored, err := s.repo.GetRefreshToken(ctx, hash)
	if err != nil || time.Now().After(stored.ExpiresAt) {
		return TokenPair{}, ErrRefreshTokenInvalid
	}

	// REUSE DETECTION (The Global Nuke):
	// If the token was already revoked, someone is trying to use a spent token.
	// We assume a breach and immediately revoke ALL of this user's active tokens.
	if stored.RevokedAt != nil {
		_ = s.repo.RevokeAllUserRefreshTokens(ctx, stored.UserID)
		return TokenPair{}, ErrRefreshTokenInvalid
	}

	_ = s.repo.RevokeRefreshToken(ctx, stored.ID)
	return s.issueTokenPair(ctx, stored.UserID)
}

// Logout revokes a refresh token. Unlike Refresh/Login, an unknown or
// already-invalid token is not an error here — logging out twice, or
// logging out with a stale token, should silently succeed from the
// caller's point of view.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	hash := auth.HashToken(refreshToken)
	stored, err := s.repo.GetRefreshToken(ctx, hash)
	if err == nil {
		_ = s.repo.RevokeRefreshToken(ctx, stored.ID)
	}
	return nil
}

// issueTokenPair mints a new JWT access token and a new random
// refresh token, persists the refresh token's hash, and returns both
// raw values to hand back to the client.
func (s *AuthService) issueTokenPair(ctx context.Context, userID uuid.UUID) (TokenPair, error) {
	access, err := s.issuer.Issue(userID)
	if err != nil {
		return TokenPair{}, err
	}

	rawRefresh, hashRefresh, err := auth.GenerateRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}

	if _, err := s.repo.CreateRefreshToken(ctx, userID, hashRefresh, time.Now().AddDate(0, 0, refreshTokenTTLDays)); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{AccessToken: access, RefreshToken: rawRefresh}, nil
}

// LoginWithGoogle takes a Google ID token from the client, verifies it,
// and either logs the user in (if they exist) or creates a new account.
func (s *AuthService) LoginWithGoogle(ctx context.Context, idToken string) (TokenPair, error) {
	payload, err := idtoken.Validate(ctx, idToken, s.googleClientID)
	if err != nil {
		return TokenPair{}, ErrInvalidCredentials
	}

	emailRaw, ok := payload.Claims["email"]
	if !ok {
		return TokenPair{}, ErrInvalidCredentials
	}
	email, ok := emailRaw.(string)
	if !ok || email == "" {
		return TokenPair{}, ErrInvalidCredentials
	}

	googleID := payload.Subject

	// First, check if the OAuth connection already exists
	user, err := s.repo.GetUserByOAuthProvider(ctx, "google", googleID)
	if err == nil {
		return s.issueTokenPair(ctx, user.ID)
	}

	// Connection doesn't exist. Check if email exists to link them.
	user, err = s.repo.GetUserByEmail(ctx, email)
	if err == nil {
		_ = s.repo.CreateOAuthConnection(ctx, user.ID, "google", googleID)
		return s.issueTokenPair(ctx, user.ID)
	}

	// Completely new user. Try to extract names from payload.
	var firstName, lastName string
	if givenNameRaw, ok := payload.Claims["given_name"]; ok {
		firstName, _ = givenNameRaw.(string)
	}
	if familyNameRaw, ok := payload.Claims["family_name"]; ok {
		lastName, _ = familyNameRaw.(string)
	}

	// Create user with null password
	user, err = s.repo.CreateUser(ctx, firstName, lastName, email, nil)
	if err != nil {
		return TokenPair{}, err
	}

	// Since Google verified the email, we mark it verified immediately
	_ = s.repo.MarkEmailVerified(ctx, user.ID)
	_ = s.repo.CreateOAuthConnection(ctx, user.ID, "google", googleID)

	return s.issueTokenPair(ctx, user.ID)
}

// GetProfile fetches the public profile of the authenticated user by their ID.
// Returns ErrUserNotFound if no user row exists for the given ID.
func (s *AuthService) GetProfile(ctx context.Context, userID uuid.UUID) (UserProfile, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return UserProfile{}, ErrUserNotFound
	}
	return UserProfile{
		ID:         user.ID,
		FirstName:  strPtrVal(user.FirstName),
		LastName:   strPtrVal(user.LastName),
		Email:      user.Email,
		IsVerified: user.EmailVerifiedAt != nil,
	}, nil
}

// strPtrVal safely dereferences a *string, returning "" if nil.
func strPtrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Ensure db is used — the import is needed for CreateUser etc. called via repo,
// but also referenced here to avoid a blank import.
var _ db.User
