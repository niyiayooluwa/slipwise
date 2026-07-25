// Package auth holds the low-level auth primitives: JWT
// issuing/verification (jwt.go), password/OTP hashing via bcrypt
// (password.go), refresh-token generation/hashing via sha256
// (tokens.go), and the HTTP middleware that enforces a valid JWT on
// protected routes (this file).
//
// This package deliberately knows nothing about HTTP request/response
// bodies, the database, or business rules like OTP expiry or attempt
// limits — those live in auth/service. This package is the toolbox;
// auth/service is what decides when and how to use it.
package auth

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

// ctxKey is an unexported type for context keys, per the standard Go
// idiom — prevents collisions with keys set by other packages using
// plain strings.
type ctxKey string

// UserIDKey is the context key RequireAuth stores the authenticated
// user's ID under. Downstream handlers read it via
// r.Context().Value(auth.UserIDKey).(uuid.UUID).
const UserIDKey ctxKey = "user_id"

// RequireAuth returns middleware that validates the Bearer JWT on
// every request and injects the resulting user ID into the request
// context. It deliberately does not check roles or permissions —
// that's a separate middleware layered on top for routes that need
// it (e.g. admin-only endpoints), so a plain authenticated route
// doesn't pay for a permissions lookup it doesn't need.
func RequireAuth(issuer *JWTIssuer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			h := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
			}
			token := strings.TrimPrefix(h, "Bearer ")

			claims, err := issuer.Verify(token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
			}

			// Store user ID in Echo's per-request store (faster than context.WithValue)
			c.Set(string(UserIDKey), claims.UserID)
			return next(c)
		}
	}
}
