// Package handler is the auth domain's HTTP layer. It decodes
// requests into model DTOs, calls AuthService, and maps domain errors
// (from service/errors.go) onto HTTP status codes and
// apitypes.ErrorResponse bodies. It holds no business logic itself —
// if you're tempted to add an if-statement here that isn't about
// decoding/encoding or error-to-status mapping, it belongs in service
// instead.
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"sportloga/internal/apitypes"
	"sportloga/internal/auth/model"
	"sportloga/internal/auth/service"

	"github.com/google/uuid"

	"github.com/labstack/echo/v5"
)

// AuthHandler exposes signup/verify/login/refresh/logout as
// echo.HandlerFuncs, backed by an AuthService.
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler builds an AuthHandler over the given AuthService.
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Signup godoc
// @Summary      Create a new account
// @Description  Creates a user with email_verified_at = null and sends
// @Description  a 6-digit OTP to the given email. The account cannot
// @Description  log in until Verify is called with that code.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body model.SignupRequest true "first name, last name, email, and password"
// @Success      201 {object} model.SignupResponse
// @Failure      400 {object} apitypes.ErrorResponse "missing required field, or password under 8 chars"
// @Failure      409 {object} apitypes.ErrorResponse "email already registered"
// @Failure      500 {object} apitypes.ErrorResponse "hashing or DB failure"
// @Router       /auth/signup [post]
func (h *AuthHandler) Signup(c *echo.Context) error {
	var req model.SignupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "invalid body"})
	}
	if req.FirstName == "" {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "first name is required"})
	}
	if req.LastName == "" {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "last name is required"})
	}
	if req.Email == "" {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "email is required"})
	}
	if len(req.Password) < 8 {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "password must be at least 8 characters"})
	}

	err := h.svc.Signup(c.Request().Context(), req.FirstName, req.LastName, req.Email, req.Password)
	switch {
	case err == nil:
		return c.JSON(http.StatusCreated, model.SignupResponse{
			Message: "account created, check your email for a verification code",
		})
	case errors.Is(err, service.ErrEmailAlreadyRegistered):
		return c.JSON(http.StatusConflict, apitypes.ErrorResponse{Error: err.Error()})
	default:
		slog.Error("auth handler error", "endpoint", "signup", "error", err)
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: "internal error"})
	}
}

// Verify godoc
// @Summary      Confirm signup OTP and activate the account
// @Description  Checks the 6-digit code against the most recent
// @Description  unused OTP for this email. On success, marks the
// @Description  account verified and returns a usable token pair —
// @Description  the caller is logged in immediately, no separate
// @Description  Login call needed. Wrong-guess attempts count even on
// @Description  failure; after 3 the code is dead regardless of
// @Description  whether the next guess would've been right.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body model.VerifyRequest true "email and OTP code"
// @Success      200 {object} model.TokenPairResponse
// @Failure      400 {object} apitypes.ErrorResponse "no active code, code expired, or code incorrect"
// @Failure      429 {object} apitypes.ErrorResponse "too many wrong attempts against this code"
// @Failure      500 {object} apitypes.ErrorResponse "DB or token-issuing failure"
// @Router       /auth/verify [post]
func (h *AuthHandler) Verify(c *echo.Context) error {
	var req model.VerifyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "invalid body"})
	}

	pair, err := h.svc.Verify(c.Request().Context(), req.Email, req.Code)
	switch {
	case err == nil:
		return c.JSON(http.StatusOK, model.TokenPairResponse{
			AccessToken:  pair.AccessToken,
			RefreshToken: pair.RefreshToken,
		})
	case errors.Is(err, service.ErrOTPNotFound), errors.Is(err, service.ErrOTPExpired), errors.Is(err, service.ErrOTPIncorrect):
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrOTPMaxAttempts):
		return c.JSON(http.StatusTooManyRequests, apitypes.ErrorResponse{Error: err.Error()})
	default:
		slog.Error("auth handler error", "endpoint", "verify", "error", err)
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: "internal error"})
	}
}

// Login godoc
// @Summary      Log in with email and password
// @Description  Returns 401 for both "no such user" and "wrong
// @Description  password" — the same error either way, deliberately,
// @Description  so a caller can't use this endpoint to enumerate which
// @Description  emails are registered. An unverified account returns
// @Description  403 separately, since at that point the password has
// @Description  already confirmed the account exists.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body model.LoginRequest true "email and password"
// @Success      200 {object} model.TokenPairResponse
// @Failure      401 {object} apitypes.ErrorResponse "invalid email or password"
// @Failure      403 {object} apitypes.ErrorResponse "email not verified"
// @Failure      500 {object} apitypes.ErrorResponse "DB or token-issuing failure"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *echo.Context) error {
	var req model.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "invalid body"})
	}

	pair, err := h.svc.Login(c.Request().Context(), req.Email, req.Password)
	switch {
	case err == nil:
		return c.JSON(http.StatusOK, model.TokenPairResponse{
			AccessToken:  pair.AccessToken,
			RefreshToken: pair.RefreshToken,
		})
	case errors.Is(err, service.ErrInvalidCredentials):
		return c.JSON(http.StatusUnauthorized, apitypes.ErrorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrOAuthAccount):
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrEmailNotVerified):
		return c.JSON(http.StatusForbidden, apitypes.ErrorResponse{Error: err.Error()})
	default:
		slog.Error("auth handler error", "endpoint", "login", "error", err)
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: "internal error"})
	}
}

// Refresh godoc
// @Summary      Rotate a refresh token for a new access/refresh pair
// @Description  The presented refresh token is revoked and a brand
// @Description  new pair is issued, even though the request otherwise
// @Description  succeeds — this bounds how long a stolen refresh
// @Description  token remains useful past the legitimate user's next
// @Description  refresh call.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body model.RefreshRequest true "refresh token to rotate"
// @Success      200 {object} model.TokenPairResponse
// @Failure      401 {object} apitypes.ErrorResponse "token invalid, revoked, or expired"
// @Failure      500 {object} apitypes.ErrorResponse "DB or token-issuing failure"
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *echo.Context) error {
	var req model.RefreshRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "invalid body"})
	}

	pair, err := h.svc.Refresh(c.Request().Context(), req.RefreshToken)
	switch {
	case err == nil:
		return c.JSON(http.StatusOK, model.TokenPairResponse{
			AccessToken:  pair.AccessToken,
			RefreshToken: pair.RefreshToken,
		})
	case errors.Is(err, service.ErrRefreshTokenInvalid):
		return c.JSON(http.StatusUnauthorized, apitypes.ErrorResponse{Error: err.Error()})
	default:
		slog.Error("auth handler error", "endpoint", "refresh", "error", err)
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: "internal error"})
	}
}

// ResendOTP godoc
// @Summary      Request a fresh signup OTP
// @Description  Self-service escape hatch for an account stuck
// @Description  mid-verification — used when the original code
// @Description  expired (10 min) or was guessed wrong 3 times. Rate
// @Description  limited to one send per 60 seconds per email so a
// @Description  caller can't hammer the mail provider or the
// @Description  recipient's inbox. The previous code is not
// @Description  explicitly revoked; it simply stops being the
// @Description  "latest" one once this issues a new one.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body model.ResendOTPRequest true "email to resend a code to"
// @Success      200 {object} apitypes.MessageResponse
// @Failure      400 {object} apitypes.ErrorResponse "account already verified"
// @Failure      404 {object} apitypes.ErrorResponse "no account for this email"
// @Failure      429 {object} apitypes.ErrorResponse "resend requested too soon after the last one"
// @Failure      500 {object} apitypes.ErrorResponse "DB or mail-provider failure"
// @Router       /auth/resend-otp [post]
func (h *AuthHandler) ResendOTP(c *echo.Context) error {
	var req model.ResendOTPRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "invalid body"})
	}

	err := h.svc.ResendOTP(c.Request().Context(), req.Email)
	switch {
	case err == nil:
		return c.JSON(http.StatusOK, apitypes.MessageResponse{
			Message: "a new code has been sent to your email",
		})
	case errors.Is(err, service.ErrUserNotFound):
		return c.JSON(http.StatusNotFound, apitypes.ErrorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrAlreadyVerified):
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrOTPCooldown):
		return c.JSON(http.StatusTooManyRequests, apitypes.ErrorResponse{Error: err.Error()})
	default:
		slog.Error("auth handler error", "endpoint", "resend-otp", "error", err)
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: "internal error"})
	}
}

// Logout godoc
// @Summary      Revoke a refresh token
// @Description  Idempotent — revoking an already-invalid or
// @Description  already-revoked token still returns 200, since from
// @Description  the caller's point of view "logged out" is true
// @Description  either way.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body model.LogoutRequest true "refresh token to revoke"
// @Success      200 {object} apitypes.MessageResponse
// @Failure      400 {object} apitypes.ErrorResponse "invalid body"
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *echo.Context) error {
	var req model.LogoutRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "invalid body"})
	}

	_ = h.svc.Logout(c.Request().Context(), req.RefreshToken)
	return c.JSON(http.StatusOK, apitypes.MessageResponse{Message: "logged out"})
}

// GoogleLogin godoc
// @Summary      Log in or sign up with Google
// @Description  Takes an ID Token from the mobile client and logs the user in, creating an account if necessary.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body model.OAuthLoginRequest true "Google ID token"
// @Success      200 {object} model.TokenPairResponse
// @Failure      400 {object} apitypes.ErrorResponse "invalid body"
// @Failure      401 {object} apitypes.ErrorResponse "invalid ID token"
// @Failure      500 {object} apitypes.ErrorResponse "internal error"
// @Router       /auth/oauth/google [post]
func (h *AuthHandler) GoogleLogin(c *echo.Context) error {
	var req model.OAuthLoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "invalid body"})
	}

	pair, err := h.svc.LoginWithGoogle(c.Request().Context(), req.IDToken)
	switch {
	case err == nil:
		return c.JSON(http.StatusOK, model.TokenPairResponse{
			AccessToken:  pair.AccessToken,
			RefreshToken: pair.RefreshToken,
		})
	case errors.Is(err, service.ErrInvalidCredentials):
		return c.JSON(http.StatusUnauthorized, apitypes.ErrorResponse{Error: err.Error()})
	default:
		slog.Error("auth handler error", "endpoint", "google-login", "error", err)
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: "internal error"})
	}
}

// Me godoc
// @Summary      Fetch the authenticated user's profile
// @Description  Returns the public profile fields of the currently logged-in user.
// @Description  Requires a valid Bearer token in the Authorization header.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} model.UserProfileResponse
// @Failure      401 {object} apitypes.ErrorResponse "missing or invalid token"
// @Failure      404 {object} apitypes.ErrorResponse "user not found"
// @Failure      500 {object} apitypes.ErrorResponse "internal error"
// @Router       /auth/me [get]
func (h *AuthHandler) Me(c *echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, apitypes.ErrorResponse{Error: "unauthorized"})
	}

	profile, err := h.svc.GetProfile(c.Request().Context(), userID)
	switch {
	case err == nil:
		return c.JSON(http.StatusOK, model.UserProfileResponse{
			ID:         profile.ID.String(),
			FirstName:  profile.FirstName,
			LastName:   profile.LastName,
			Username:   profile.Username,
			Email:      profile.Email,
			IsVerified: profile.IsVerified,
		})
	case errors.Is(err, service.ErrUserNotFound):
		return c.JSON(http.StatusNotFound, apitypes.ErrorResponse{Error: err.Error()})
	default:
		slog.Error("auth handler error", "endpoint", "me", "error", err)
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: "internal error"})
	}
}

// ForgotPassword godoc
// @Summary      Request a password reset OTP
// @Description  Sends a 6-digit OTP to the user's email if they exist.
// @Description  Returns 200 even if the user doesn't exist to prevent email enumeration.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body model.ForgotPasswordRequest true "email to send OTP to"
// @Success      200 {object} apitypes.MessageResponse
// @Failure      400 {object} apitypes.ErrorResponse "invalid body"
// @Failure      429 {object} apitypes.ErrorResponse "requested too soon after the last one"
// @Failure      500 {object} apitypes.ErrorResponse "DB or mail-provider failure"
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *echo.Context) error {
	var req model.ForgotPasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "invalid body"})
	}
	if req.Email == "" {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "email is required"})
	}

	err := h.svc.ForgotPassword(c.Request().Context(), req.Email)
	switch {
	case err == nil:
		return c.JSON(http.StatusOK, apitypes.MessageResponse{
			Message: "if the email exists, a password reset code has been sent",
		})
	case errors.Is(err, service.ErrOTPCooldown):
		return c.JSON(http.StatusTooManyRequests, apitypes.ErrorResponse{Error: err.Error()})
	default:
		slog.Error("auth handler error", "endpoint", "forgot-password", "error", err)
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: "internal error"})
	}
}

// ResetPassword godoc
// @Summary      Reset password using OTP
// @Description  Validates the reset OTP and updates the user's password.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body model.ResetPasswordRequest true "email, code, and new password"
// @Success      200 {object} apitypes.MessageResponse
// @Failure      400 {object} apitypes.ErrorResponse "invalid body or OTP incorrect/expired"
// @Failure      429 {object} apitypes.ErrorResponse "too many wrong attempts against this code"
// @Failure      500 {object} apitypes.ErrorResponse "internal error"
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *echo.Context) error {
	var req model.ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "invalid body"})
	}
	if req.Email == "" || req.Code == "" || req.NewPassword == "" {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "email, code, and new_password are required"})
	}
	if len(req.NewPassword) < 8 {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "password must be at least 8 characters"})
	}

	err := h.svc.ResetPassword(c.Request().Context(), req.Email, req.Code, req.NewPassword)
	switch {
	case err == nil:
		return c.JSON(http.StatusOK, apitypes.MessageResponse{
			Message: "password updated successfully",
		})
	case errors.Is(err, service.ErrOTPNotFound), errors.Is(err, service.ErrOTPExpired), errors.Is(err, service.ErrOTPIncorrect):
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrOTPMaxAttempts):
		return c.JSON(http.StatusTooManyRequests, apitypes.ErrorResponse{Error: err.Error()})
	default:
		slog.Error("auth handler error", "endpoint", "reset-password", "error", err)
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: "internal error"})
	}
}

// UpdateProfile godoc
// @Summary      Update user profile
// @Description  Updates the authenticated user's profile fields.
// @Description  Requires a valid Bearer token in the Authorization header.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body model.UpdateProfileRequest true "profile fields to update"
// @Success      200 {object} model.UserProfileResponse
// @Failure      400 {object} apitypes.ErrorResponse "invalid body"
// @Failure      401 {object} apitypes.ErrorResponse "unauthorized"
// @Failure      500 {object} apitypes.ErrorResponse "internal error"
// @Router       /auth/me [patch]
func (h *AuthHandler) UpdateProfile(c *echo.Context) error {
	var req model.UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "invalid body"})
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return c.JSON(http.StatusUnauthorized, apitypes.ErrorResponse{Error: "unauthorized"})
	}

	profile, err := h.svc.UpdateProfile(c.Request().Context(), userID, req.FirstName, req.LastName, req.Username)
	if err != nil {
		slog.Error("auth handler error", "endpoint", "update-profile", "error", err)
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: "internal error"})
	}

	return c.JSON(http.StatusOK, model.UserProfileResponse{
		ID:         profile.ID.String(),
		FirstName:  profile.FirstName,
		LastName:   profile.LastName,
		Username:   profile.Username,
		Email:      profile.Email,
		IsVerified: profile.IsVerified,
	})
}

// CheckUsername godoc
// @Summary      Check username availability
// @Description  Checks if a given username is available to be claimed.
// @Tags         auth
// @Produce      json
// @Param        q query string true "username to check"
// @Success      200 {object} model.CheckUsernameResponse
// @Failure      400 {object} apitypes.ErrorResponse "missing query parameter"
// @Failure      500 {object} apitypes.ErrorResponse "internal error"
// @Router       /auth/check-username [get]
func (h *AuthHandler) CheckUsername(c *echo.Context) error {
	username := c.QueryParam("q")
	if username == "" {
		return c.JSON(http.StatusBadRequest, apitypes.ErrorResponse{Error: "missing 'q' query parameter"})
	}

	available, err := h.svc.CheckUsername(c.Request().Context(), username)
	if err != nil {
		slog.Error("auth handler error", "endpoint", "check-username", "error", err)
		return c.JSON(http.StatusInternalServerError, apitypes.ErrorResponse{Error: "internal error"})
	}

	return c.JSON(http.StatusOK, model.CheckUsernameResponse{
		Available: available,
	})
}
