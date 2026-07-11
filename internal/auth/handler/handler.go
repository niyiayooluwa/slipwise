// Package handler is the auth domain's HTTP layer. It decodes
// requests into model DTOs, calls AuthService, and maps domain errors
// (from service/errors.go) onto HTTP status codes and
// apitypes.ErrorResponse bodies. It holds no business logic itself —
// if you're tempted to add an if-statement here that isn't about
// decoding/encoding or error-to-status mapping, it belongs in service
// instead.
package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"sportloga/internal/apitypes"
	"sportloga/internal/auth/model"
	"sportloga/internal/auth/service"
	"sportloga/internal/response"
)

// AuthHandler exposes signup/verify/login/refresh/logout as
// http.HandlerFuncs, backed by an AuthService.
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
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req model.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.FirstName == "" || req.LastName == "" || req.Email == "" || len(req.Password) < 8 {
		response.WriteError(w, http.StatusBadRequest, "first name, last name, and email required, password min 8 chars")
		return
	}

	err := h.svc.Signup(r.Context(), req.FirstName, req.LastName, req.Email, req.Password)
	switch {
	case err == nil:
		response.WriteJSON(w, http.StatusCreated, model.SignupResponse{
			Message: "account created, check your email for a verification code",
		})
	case errors.Is(err, service.ErrEmailAlreadyRegistered):
		response.WriteError(w, http.StatusConflict, err.Error())
	default:
		slog.Error("auth handler error", "endpoint", "signup", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "internal error")
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
func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req model.VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	pair, err := h.svc.Verify(r.Context(), req.Email, req.Code)
	switch {
	case err == nil:
		response.WriteJSON(w, http.StatusOK, model.TokenPairResponse{
			AccessToken:  pair.AccessToken,
			RefreshToken: pair.RefreshToken,
		})
	case errors.Is(err, service.ErrOTPNotFound), errors.Is(err, service.ErrOTPExpired), errors.Is(err, service.ErrOTPIncorrect):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrOTPMaxAttempts):
		response.WriteError(w, http.StatusTooManyRequests, err.Error())
	default:
		slog.Error("auth handler error", "endpoint", "verify", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "internal error")
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
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	pair, err := h.svc.Login(r.Context(), req.Email, req.Password)
	switch {
	case err == nil:
		response.WriteJSON(w, http.StatusOK, model.TokenPairResponse{
			AccessToken:  pair.AccessToken,
			RefreshToken: pair.RefreshToken,
		})
	case errors.Is(err, service.ErrInvalidCredentials):
		response.WriteError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, service.ErrEmailNotVerified):
		response.WriteError(w, http.StatusForbidden, err.Error())
	default:
		slog.Error("auth handler error", "endpoint", "login", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "internal error")
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
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	pair, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	switch {
	case err == nil:
		response.WriteJSON(w, http.StatusOK, model.TokenPairResponse{
			AccessToken:  pair.AccessToken,
			RefreshToken: pair.RefreshToken,
		})
	case errors.Is(err, service.ErrRefreshTokenInvalid):
		response.WriteError(w, http.StatusUnauthorized, err.Error())
	default:
		slog.Error("auth handler error", "endpoint", "refresh", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "internal error")
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
func (h *AuthHandler) ResendOTP(w http.ResponseWriter, r *http.Request) {
	var req model.ResendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	err := h.svc.ResendOTP(r.Context(), req.Email)
	switch {
	case err == nil:
		response.WriteJSON(w, http.StatusOK, apitypes.MessageResponse{
			Message: "a new code has been sent to your email",
		})
	case errors.Is(err, service.ErrUserNotFound):
		response.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrAlreadyVerified):
		response.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrOTPCooldown):
		response.WriteError(w, http.StatusTooManyRequests, err.Error())
	default:
		slog.Error("auth handler error", "endpoint", "resend-otp", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "internal error")
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
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req model.LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	_ = h.svc.Logout(r.Context(), req.RefreshToken)
	response.WriteJSON(w, http.StatusOK, apitypes.MessageResponse{Message: "logged out"})
}