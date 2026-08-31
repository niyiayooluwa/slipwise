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

// ctxKey is a private type for context keys to avoid collisions with other packages.
type ctxKey string

// UserIDKey is the context key under which the authenticated user's UUID is stored.
// Downstream handlers access this via `c.Get(string(auth.UserIDKey)).(uuid.UUID)`.
const UserIDKey ctxKey = "user_id"

// RequireAuth returns an Echo middleware that enforces a valid Bearer JWT on protected routes.
//
// Workflow:
// 1. Reads the `Authorization: Bearer <token>` header.
// 2. Verifies the signature, expiry, and HMAC algorithm via `issuer.Verify`.
// 3. Extracts the user's `uuid.UUID` from the JWT claims and saves it to Echo's per-request context (`c.Set(...)`).
// 4. Calls the next handler in the chain.
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

			// Store user ID in Echo's per-request store (faster than standard context.WithValue allocations)
			c.Set(string(UserIDKey), claims.UserID)
			return next(c)
		}
	}
}
