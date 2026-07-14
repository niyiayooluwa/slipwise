package admin

import (
	"context"
	"net/http"

	"sportloga/internal/auth"
	"sportloga/internal/db/generated"

	"github.com/google/uuid"
)

// DB represents the query interface needed by the admin middleware.
type DB interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (generated.User, error)
}

// RequireAdmin creates a middleware that ensures the authenticated user
// has the IsAdmin flag set to true in the database.
// It MUST be placed after auth.RequireAuth in the router chain.
func RequireAdmin(queries DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Get the UserID from the context (injected by auth.RequireAuth)
			userIDRaw := r.Context().Value(auth.UserIDKey)
			if userIDRaw == nil {
				// This happens if someone forgot to put auth.RequireAuth before this middleware
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			userID, ok := userIDRaw.(uuid.UUID)
			if !ok {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// 2. Fetch the user from the database to check their latest role
			user, err := queries.GetUserByID(r.Context(), userID)
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// 3. Check the IsAdmin flag we added in the SQL migration!
			if !user.IsAdmin {
				http.Error(w, `{"error":"forbidden: admin access required"}`, http.StatusForbidden)
				return
			}

			// 4. Success! Let them through to the admin endpoint
			next.ServeHTTP(w, r)
		})
	}
}
