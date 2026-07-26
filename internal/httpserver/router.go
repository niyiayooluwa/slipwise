// Package httpserver holds cross-domain HTTP plumbing: the Echo router
// and route wiring for every domain.
package httpserver

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"sportloga/internal/auth"
	authhandler "sportloga/internal/auth/handler"

	"github.com/go-chi/httprate"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
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

// NewRouter builds the full Echo router. trustedProxyCIDRs controls client-IP
// resolution: empty → direct internet (use RemoteAddr); populated → read
// X-Forwarded-For, trusting only the listed CIDR ranges as proxy hops. No
// code change is needed when switching between deployment topologies — only
// the TRUSTED_PROXY_CIDRS env var needs updating.
func NewRouter(h Handlers, jwtIssuer *auth.JWTIssuer, allowedOrigins []string, trustedProxyCIDRs []string) *echo.Echo {
	e := echo.New()
	//e.HideBanner = true
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

	e.GET("/swagger/*", echo.WrapHandler(httpSwagger.WrapHandler))

	authGroup := e.Group("/auth")
	mountAuthRoutes(authGroup, h.Auth, e.IPExtractor)

	// Protected routes, once a domain needs them:
	// protectedGroup := e.Group("")
	// protectedGroup.Use(auth.RequireAuth(jwtIssuer))
	// mountRealtimeRoutes(protectedGroup.Group("/realtime"), h.Realtime)

	return e
}

// mountAuthRoutes registers all /auth endpoints.
// The IPExtractor is passed through so clientIPKey uses the same
// proxy-aware IP resolution as the rest of the server.
func mountAuthRoutes(g *echo.Group, h *authhandler.AuthHandler, extractor echo.IPExtractor) {
	loginRateLimit := echo.WrapMiddleware(
		httprate.LimitBy(5, time.Minute, clientIPKey(extractor)),
	)

	g.POST("/signup", h.Signup)
	g.POST("/verify", h.Verify)
	g.POST("/resend-otp", h.ResendOTP)
	g.POST("/login", h.Login, loginRateLimit)
	g.POST("/oauth/google", h.GoogleLogin, loginRateLimit)
	g.POST("/refresh", h.Refresh)
	g.POST("/logout", h.Logout)
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
