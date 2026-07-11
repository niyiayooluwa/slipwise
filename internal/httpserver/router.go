package httpserver

import (
	"sportloga/internal/auth"
	authhandler "sportloga/internal/auth/handler"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Handlers bundles every domain's handler so NewRouter takes one
// argument instead of growing a new parameter every time a module
// (realtime, notifications, betting, ...) gets its own handler.
type Handlers struct {
	Auth *authhandler.AuthHandler
	// Realtime      *realtimehandler.RealtimeHandler
	// Notifications *notificationshandler.NotificationsHandler
}

// NewRouter builds the full chi router: base middleware, swagger UI,
// and every domain's routes mounted under its own prefix. Route
// wiring lives here and only here — main.go should never call
// r.Post/r.Get directly.
func NewRouter(h Handlers, jwtIssuer *auth.JWTIssuer) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/auth", authRoutes(h.Auth))

	// Protected routes, once a domain needs them:
	// r.Group(func(r chi.Router) {
	// 	r.Use(auth.RequireAuth(jwtIssuer))
	// 	r.Route("/realtime", realtimeRoutes(h.Realtime))
	// })

	return r
}

// authRoutes mounts every /auth endpoint. Kept in its own function
// (rather than inline in NewRouter) so each domain's route list is
// easy to find and diff independently as the API grows.
func authRoutes(h *authhandler.AuthHandler) func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/signup", h.Signup)
		r.Post("/verify", h.Verify)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)
	}
}
