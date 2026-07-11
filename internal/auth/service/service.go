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
)

// otpTTL is how long a signup OTP stays valid after issuance.
const otpTTL = 10 * time.Minute

// otpMaxAttempts is how many wrong guesses are allowed against a
// single OTP before it's dead and a new one must be requested.
const otpMaxAttempts = 3

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

// AuthService implements signup/verify/login/refresh/logout. Construct
// with NewAuthService; all fields are unexported to keep the
// dependency list explicit at the constructor rather than settable ad
// hoc.
type AuthService struct {
	repo   repository.AuthRepository
	issuer *auth.JWTIssuer
	mailer Mailer
}

// NewAuthService wires an AuthService from its three dependencies: the
// persistence layer, the JWT issuer, and the email sender.
func NewAuthService(repo repository.AuthRepository, issuer *auth.JWTIssuer, mailer Mailer) *AuthService {
	return &AuthService{repo: repo, issuer: issuer, mailer: mailer}
}

// Signup creates a new, unverified user and sends a signup OTP to
// email. The user row is created immediately (with
// email_verified_at = null) rather than deferred until Verify
// succeeds — this trades "a signup that's abandoned mid-OTP leaves a
// dangling unverified row" for "an OTP always has a real user_id to
// belong to." Returns ErrEmailAlreadyRegistered if email is taken.
func (s *AuthService) Signup(ctx context.Context, email, password string) error {
	if _, err := s.repo.GetUserByEmail(ctx, email); err == nil {
		return ErrEmailAlreadyRegistered
	}

	pwHash, err := auth.HashSecret(password)
	if err != nil {
		return err
	}

	user, err := s.repo.CreateUser(ctx, email, pwHash)
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

// Login checks password and verification status, then issues a fresh
// token pair. Returns ErrInvalidCredentials for both "no such user"
// and "wrong password" (see errors.go for why), and
// ErrEmailNotVerified separately since that's not a secret — the
// account's existence is already implied by the correct password.
func (s *AuthService) Login(ctx context.Context, email, password string) (TokenPair, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil || !auth.CheckSecret(user.PasswordHash, password) {
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
	if err != nil || stored.RevokedAt != nil || time.Now().After(stored.ExpiresAt) {
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
