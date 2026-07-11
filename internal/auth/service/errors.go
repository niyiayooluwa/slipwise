package service

import "errors"

// ErrEmailAlreadyRegistered is returned by Signup when the email is
// already tied to an account. Maps to HTTP 409 in the handler.
var ErrEmailAlreadyRegistered = errors.New("email already registered")

// ErrInvalidCredentials is returned by Login for both "no such user"
// and "wrong password". Deliberately the same error for both cases —
// returning a different one would let a caller enumerate which emails
// are registered. Maps to HTTP 401.
var ErrInvalidCredentials = errors.New("invalid email or password")

// ErrEmailNotVerified is returned by Login when the account exists and
// the password is correct, but signup OTP verification was never
// completed. Maps to HTTP 403.
var ErrEmailNotVerified = errors.New("email not verified")

// ErrOTPNotFound is returned by Verify when there's no live (unused)
// OTP for the given email/purpose — either none was ever requested or
// a previous one was already consumed. Maps to HTTP 400.
var ErrOTPNotFound = errors.New("no active code for this email")

// ErrOTPExpired is returned by Verify when a matching OTP exists but
// its 10-minute window has passed. Maps to HTTP 400.
var ErrOTPExpired = errors.New("code expired, request a new one")

// ErrOTPMaxAttempts is returned by Verify once 3 wrong guesses have
// been made against a given OTP — it is dead even if the *next* guess
// would've been correct, forcing a fresh code rather than allowing
// unlimited retries against one code. Maps to HTTP 429.
var ErrOTPMaxAttempts = errors.New("too many attempts, request a new code")

// ErrOTPIncorrect is returned by Verify when the code doesn't match.
// Maps to HTTP 400.
var ErrOTPIncorrect = errors.New("incorrect code")

// ErrRefreshTokenInvalid is returned by Refresh/Logout when the token
// doesn't exist, was already revoked, or has expired. Same error for
// all three cases for the same anti-enumeration reason as
// ErrInvalidCredentials. Maps to HTTP 401.
var ErrRefreshTokenInvalid = errors.New("invalid or expired refresh token")

// ErrUserNotFound is returned by ResendOTP when no account exists for
// the given email. Unlike Login, this isn't hidden behind a generic
// message — Signup already reveals account existence via 409, so
// there's no new enumeration surface to protect here. Maps to HTTP 404.
var ErrUserNotFound = errors.New("no account found for this email")

// ErrAlreadyVerified is returned by ResendOTP when the account exists
// but has already completed signup verification — the caller should
// be using Login, not requesting another code. Maps to HTTP 400.
var ErrAlreadyVerified = errors.New("account already verified, try logging in")

// ErrOTPCooldown is returned by ResendOTP when a live OTP was issued
// too recently. Prevents a caller from hammering the email provider
// (and a user's inbox) with repeated sends. Maps to HTTP 429.
var ErrOTPCooldown = errors.New("please wait before requesting another code")