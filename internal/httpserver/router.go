// Package httpserver holds cross-domain HTTP plumbing: the Echo router
// and route wiring for every domain.
package httpserver

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	adminhandler "slipwise/internal/admin/handler"
	"slipwise/internal/auth"
	authhandler "slipwise/internal/auth/handler"
	bettinghandler "slipwise/internal/betting/handler"
	"slipwise/internal/worker"

	"github.com/go-chi/httprate"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Handlers bundles every domain's handler so NewRouter takes one
// argument instead of growing a new parameter every time a module
// (realtime, notifications, betting, ...) gets its own handler.
type Handlers struct {
	Auth    *authhandler.AuthHandler
	Betting *bettinghandler.BettingHandler
	Poller  *worker.Poller
	// Admin handler + its pre-built RequireAdmin middleware.
	// Both are wired in main.go (where queries lives), keeping the router
	// completely decoupled from the DB layer.
	Admin           *adminhandler.AdminHandler
	AdminMiddleware echo.MiddlewareFunc
}

// NewRouter builds the complete Echo HTTP application router.
//
// Middleware Pipeline:
// 1. RequestID: Injects a unique X-Request-ID header on every response for tracing across log systems.
// 2. RequestLogger: Structured JSON access logs with HTTP method, latency, and status code.
// 3. Recover: Intercepts unhandled panics and returns clean 500 responses without crashing the binary.
// 4. CORS: Cross-Origin Resource Sharing rules for mobile/web frontends.
// 5. Proxy-Aware IP Extractor: Resolves true client IPs across Cloudflare / reverse proxies.
func NewRouter(h Handlers, jwtIssuer *auth.JWTIssuer, allowedOrigins []string, trustedProxyCIDRs []string, cronSecret string) *echo.Echo {
	e := echo.New()
	e.IPExtractor = buildIPExtractor(trustedProxyCIDRs)

	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Interactive OpenAPI / Swagger UI documentation endpoint: GET /swagger/index.html
	e.GET("/swagger/*", echo.WrapHandler(httpSwagger.WrapHandler))

	// Health check endpoint (Public) — Used by Railway/Docker uptime monitors and load balancers.
	e.GET("/health", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Cron endpoint to manually trigger the settlement poller (useful for serverless setups).
	e.POST("/internal/cron/settle", func(c *echo.Context) error {
		if cronSecret != "" && c.Request().Header.Get("X-Cron-Secret") != cronSecret {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
		}
		if h.Poller != nil {
			h.Poller.RunOnce(c.Request().Context())
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "settlement_triggered"})
	})

	authGroup := e.Group("/auth")
	mountAuthRoutes(authGroup, h.Auth, e.IPExtractor, jwtIssuer)

	// Protected routes behind JWT auth
	protectedGroup := e.Group("")
	protectedGroup.Use(auth.RequireAuth(jwtIssuer))

	userGroup := protectedGroup.Group("/v1/users")
	userGroup.GET("/me/stats", h.Betting.GetStats, CacheControl(10))

	mountBettingRoutes(protectedGroup.Group("/v1/tickets"), h.Betting)

	// Admin routes — JWT auth (above) + RequireAdmin middleware (pre-built in main.go).
	// The router has zero knowledge of the DB; it just applies the pre-wired middleware.
	adminGroup := protectedGroup.Group("/v1/admin", h.AdminMiddleware)
	adminGroup.GET("/dashboard", h.Admin.GetDashboardStats)
	adminGroup.GET("/users", h.Admin.GetUsers)

	return e
}

// mountAuthRoutes registers all /auth endpoints.
// The IPExtractor is passed through so clientIPKey uses the same
// proxy-aware IP resolution as the rest of the server.
func mountAuthRoutes(g *echo.Group, h *authhandler.AuthHandler, extractor echo.IPExtractor, jwtIssuer *auth.JWTIssuer) {
	loginRateLimit := echo.WrapMiddleware(
		httprate.LimitBy(5, time.Minute, clientIPKey(extractor)),
	)

	g.POST("/signup", h.Signup, loginRateLimit)
	g.POST("/verify", h.Verify)
	g.POST("/resend-otp", h.ResendOTP, loginRateLimit)
	g.POST("/login", h.Login, loginRateLimit)
	g.POST("/oauth/google", h.GoogleLogin, loginRateLimit)
	g.POST("/forgot-password", h.ForgotPassword)
	g.POST("/reset-password", h.ResetPassword)
	g.GET("/check-username", h.CheckUsername, loginRateLimit)
	g.POST("/refresh", h.Refresh)
	g.POST("/logout", h.Logout)

	// Protected auth routes — require a valid JWT.
	protected := g.Group("", auth.RequireAuth(jwtIssuer))
	protected.GET("/me", h.Me)
	protected.PATCH("/me", h.UpdateProfile)
	protected.POST("/feedback", h.SubmitFeedback, loginRateLimit)
	protected.POST("/devices", h.RegisterDevice)
}

// mountBettingRoutes registers all /v1/tickets endpoints.
// All routes here are behind the JWT auth middleware applied at the group level.
func mountBettingRoutes(g *echo.Group, h *bettinghandler.BettingHandler) {
	g.POST("/preview", h.Preview)
	g.POST("/track", h.Track)
	g.GET("", h.GetHistory, CacheControl(10))
	g.GET("/archived", h.GetArchivedHistory, CacheControl(10))
	g.POST("/archive", h.BulkArchive)
	g.POST("/unarchive", h.BulkUnarchive)
	g.POST("/delete", h.BulkDelete)
	g.GET("/:id", h.GetTicketDetails)
	g.PATCH("/:id", h.UpdateTicket)
	g.DELETE("/:id", h.DeleteTicket)
}

// buildIPExtractor returns the correct Echo IPExtractor for the given
// list of trusted-proxy CIDR strings.
//   - Empty list → ExtractIPDirect: uses r.RemoteAddr only, ignores headers.
//     Safe for servers directly on the internet.
//   - Non-empty list → ExtractIPFromXFFHeader with a TrustIPRange option per
//     CIDR. Echo walks X-Forwarded-For right-to-left, skipping trusted hops,
//     and returns the first untrusted (client) IP.
//     Safe behind Nginx, Caddy, ALB, or Cloudflare — controlled entirely by
//     the TRUSTED_PROXY_CIDRS env var, no code change needed.
func buildIPExtractor(cidrs []string) echo.IPExtractor {
	if len(cidrs) == 0 {
		return echo.ExtractIPDirect()
	}
	opts := make([]echo.TrustOption, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(strings.TrimSpace(cidr))
		if err != nil {
			// Log malformed CIDRs at startup but don't crash — fall back to
			// treating them as absent so the server still starts.
			slog.Warn("TRUSTED_PROXY_CIDRS: skipping invalid CIDR", "cidr", cidr, "error", err)
			continue
		}
		opts = append(opts, echo.TrustIPRange(network))
	}
	if len(opts) == 0 {
		return echo.ExtractIPDirect()
	}
	return echo.ExtractIPFromXFFHeader(opts...)
}

// clientIPKey returns an httprate KeyFunc that uses the same IPExtractor as
// the Echo instance, so rate-limiting always sees the same client IP as the
// rest of the request pipeline — whether direct or behind a proxy.
func clientIPKey(extractor echo.IPExtractor) httprate.KeyFunc {
	return func(r *http.Request) (string, error) {
		ip := extractor(r)
		// Canonicalize IPv6 by /64 prefix so a client rotating SLAAC
		// addresses within the same delegation can't escape the rate limit.
		return httprate.CanonicalizeIP(ip), nil
	}
}

// CacheControl injects Cache-Control HTTP headers into responses.
// This prevents the frontend from spamming read-heavy endpoints.
func CacheControl(maxAgeSeconds int) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Response().Header().Set("Cache-Control", fmt.Sprintf("private, max-age=%d", maxAgeSeconds))
			return next(c)
		}
	}
}
