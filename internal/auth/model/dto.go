// Package model holds the auth domain's request/response DTOs.
//
// These are deliberately separate from the sqlc-generated structs in
// internal/db — the DB schema is free to gain/rename columns without
// silently changing the public API shape. Every type here is what
// actually crosses the HTTP boundary, and every one of them is what
// swag documents (never annotate db.User etc. directly).
package model

// SignupRequest is the body for POST /auth/signup.
type SignupRequest struct {
	Username string `json:"username" example:"johndoe99"`
	// Email is the account's login identifier. Must be unique.
	Email string `json:"email" example:"user@example.com"`
	// Password is the plaintext password from the client; hashed
	// server-side before storage, never stored or logged as-is.
	Password string `json:"password" example:"correct-horse-battery-staple"`
} // @name SignupRequest

// SignupResponse confirms account creation and that a verification
// email was sent. It intentionally carries no tokens — the account
// isn't usable until Verify succeeds.
type SignupResponse struct {
	// Message is a human-readable confirmation for the frontend to display.
	Message string `json:"message" example:"account created, check your email for a verification code"`
} // @name SignupResponse

// VerifyRequest is the body for POST /auth/verify. Confirms the OTP
// sent at signup and activates the account.
type VerifyRequest struct {
	// Email identifies which account's OTP to check.
	Email string `json:"email" example:"user@example.com"`
	// Code is the 6-digit OTP the user received by email.
	Code string `json:"code" example:"482913"`
} // @name VerifyRequest

// ResendOTPRequest is the body for POST /auth/resend-otp. Requests a
// fresh signup OTP for an account that hasn't completed verification
// yet — e.g. because the original code expired or was guessed wrong
// 3 times.
type ResendOTPRequest struct {
	// Email identifies which account to send a fresh code to.
	Email string `json:"email" example:"user@example.com"`
} // @name ResendOTPRequest

// LoginRequest is the body for POST /auth/login.
type LoginRequest struct {
	// Email is the account's login identifier.
	Email string `json:"email" example:"user@example.com"`
	// Password is the plaintext password to check against the stored hash.
	Password string `json:"password" example:"correct-horse-battery-staple"`
} // @name LoginRequest

// TokenPairResponse is returned by Verify, Login, and Refresh — anywhere
// the client receives a fresh, usable credential pair.
type TokenPairResponse struct {
	// AccessToken is a short-lived (15 min) JWT sent as a Bearer token
	// on subsequent authenticated requests.
	AccessToken string `json:"access_token"`
	// RefreshToken is a long-lived (30 day) opaque token used only
	// against POST /auth/refresh to mint a new pair. It is rotated
	// (revoked and replaced) on every use.
	RefreshToken string `json:"refresh_token"`
} // @name TokenPairResponse

// RefreshRequest is the body for POST /auth/refresh.
type RefreshRequest struct {
	// RefreshToken is the token issued by a previous login/verify/refresh call.
	RefreshToken string `json:"refresh_token"`
} // @name RefreshRequest

// LogoutRequest is the body for POST /auth/logout. Reuses the refresh
// token shape since logout just revokes it server-side.
type LogoutRequest struct {
	// RefreshToken is the token to revoke.
	RefreshToken string `json:"refresh_token"`
} // @name LogoutRequest

// Note: generic MessageResponse and ErrorResponse live in
// internal/apitypes, not here — they're cross-domain shapes, not
// auth-specific ones. See that package for both.

// OAuthLoginRequest contains the payload required to login/signup via an OAuth provider.
type OAuthLoginRequest struct {
	// IDToken is the JWT issued by the provider (e.g., Google or Apple).
	IDToken string `json:"id_token" validate:"required"`
} // @name OAuthLoginRequest

// UserProfileResponse is returned by GET /auth/me.
// It contains the public-facing fields of the authenticated user.
type UserProfileResponse struct {
	// ID is the user's unique identifier.
	ID string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// Username is the user's handle. Optional.
	Username *string `json:"username" example:"johndoe99"`
	// Email is the user's email address.
	Email string `json:"email" example:"user@example.com"`
	// IsVerified indicates whether the user has confirmed their email.
	IsVerified bool `json:"is_verified" example:"true"`
} // @name UserProfileResponse

// ForgotPasswordRequest is the body for POST /auth/forgot-password.
type ForgotPasswordRequest struct {
	Email string `json:"email" example:"user@example.com"`
} // @name ForgotPasswordRequest

// ResetPasswordRequest is the body for POST /auth/reset-password.
type ResetPasswordRequest struct {
	Email       string `json:"email" example:"user@example.com"`
	Code        string `json:"code" example:"123456"`
	NewPassword string `json:"new_password" example:"new-secure-password"`
} // @name ResetPasswordRequest

// UpdateProfileRequest is the body for PATCH /auth/me.
// All fields are optional.
type UpdateProfileRequest struct {
	Username *string `json:"username" example:"johndoe99"`
} // @name UpdateProfileRequest

// CheckUsernameResponse is the payload for the check username availability endpoint.
type CheckUsernameResponse struct {
	// Available is true if the username is not taken.
	Available bool `json:"available" example:"true"`
} // @name CheckUsernameResponse

type FeedbackRequest struct {
	Feedback string `json:"feedback" validate:"required,min=10,max=2000"`
}
