// Package admin provides administrative HTTP middleware and handlers.
package admin

import (
	"context"
	"net/http"

	"slipwise/internal/auth"
	generated "slipwise/internal/db/generated"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// DB represents the query interface needed by the admin middleware.
type DB interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (generated.User, error)
}

// RequireAdmin creates a middleware that ensures the authenticated user
// has the IsAdmin flag set to true in the database.
// It MUST be placed after auth.RequireAuth in the router chain.
func RequireAdmin(queries DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// 1. Get the UserID from the context (injected by auth.RequireAuth)
			userIDRaw := c.Get(string(auth.UserIDKey))
			if userIDRaw == nil {
				// This happens if someone forgot to put auth.RequireAuth before this middleware
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}

			userID, ok := userIDRaw.(uuid.UUID)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}

			// 2. Fetch the user from the database to check their latest role
			user, err := queries.GetUserByID(c.Request().Context(), userID)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}

			// 3. Check the IsAdmin flag we added in the SQL migration!
			if !user.IsAdmin {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden: admin access required"})
			}

			// 4. Success! Let them through to the admin endpoint
			return next(c)
		}
	}
}
