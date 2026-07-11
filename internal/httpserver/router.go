// Package httpserver holds cross-domain HTTP plumbing: the chi router
// and route wiring for every domain. Shared response helpers
// (WriteJSON/WriteError) live in internal/response instead of here —
// this package imports domain handler packages to mount their routes,
// so it can never also be something a handler package imports, or
// you get an import cycle.
package httpserver

import (
	"time"

	"sportloga/internal/auth"
	authhandler "sportloga/internal/auth/handler"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
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

// NewRouter builds the full chi router: base middleware, CORS,
// swagger UI, and every domain's routes mounted under its own prefix.
// Route wiring lives here and only here — main.go should never call
// r.Post/r.Get directly.
func NewRouter(h Handlers, jwtIssuer *auth.JWTIssuer, allowedOrigins []string) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// AllowCredentials is deliberately false: pairing it with a "*"
	// origin (the local-dev default) is invalid per the CORS spec and
	// browsers reject it outright. Once AllowedOrigins is locked down
	// to real frontend origins (not "*"), this can flip to true if
	// cookie-based auth is ever added — Bearer tokens in headers don't
	// need it.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

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
		r.Post("/resend-otp", h.ResendOTP)

		// Login gets its own rate limit, separate from the rest of
		// auth: it's the one endpoint where a wrong guess is
		// meaningful (a password check) and cheap to spam, unlike
		// signup/verify/resend which already have their own limits
		// (unique email, OTP attempt count, resend cooldown). 5
		// requests/minute per IP is generous for a real user
		// mistyping a password, punishing for a brute-force script.
		r.With(httprate.LimitByIP(5, time.Minute)).Post("/login", h.Login)

		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)
	}
}
