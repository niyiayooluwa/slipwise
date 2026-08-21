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

	"slipwise/internal/auth"
	authhandler "slipwise/internal/auth/handler"
	bettinghandler "slipwise/internal/betting/handler"

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

	// Health check — unauthenticated, used by load balancers and uptime monitors.
	e.GET("/health", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Debug endpoint to test outbound connectivity (remove in production)
e.GET("/debug/network", func(c *echo.Context) error {
    client := &http.Client{Timeout: 5 * time.Second}
    
    var result strings.Builder
    
    // Test Cloudflare Worker
    result.WriteString("Testing Cloudflare Worker...\n")
    start := time.Now()
    resp, err := client.Get("https://live-events.aytholu.workers.dev/ticket?code=Jycm08")
    if err != nil {
        result.WriteString(fmt.Sprintf("Worker Error: %v\n", err))
    } else {
        result.WriteString(fmt.Sprintf("Worker Status: %d\n", resp.StatusCode))
        resp.Body.Close()
    }
    result.WriteString(fmt.Sprintf("Worker Time: %v\n\n", time.Since(start)))
    
    // Test Sportybet Direct
    result.WriteString("Testing Sportybet Direct...\n")
    start = time.Now()
    resp2, err2 := client.Get("https://www.sportybet.com")
    if err2 != nil {
        result.WriteString(fmt.Sprintf("Sportybet Error: %v\n", err2))
    } else {
        result.WriteString(fmt.Sprintf("Sportybet Status: %d\n", resp2.StatusCode))
        resp2.Body.Close()
    }
    result.WriteString(fmt.Sprintf("Sportybet Time: %v\n\n", time.Since(start)))
    
    // Test Google (control)
    result.WriteString("Testing Google (control)...\n")
    start = time.Now()
    resp3, err3 := client.Get("https://google.com")
    if err3 != nil {
        result.WriteString(fmt.Sprintf("Google Error: %v\n", err3))
    } else {
        result.WriteString(fmt.Sprintf("Google Status: %d\n", resp3.StatusCode))
        resp3.Body.Close()
    }
    result.WriteString(fmt.Sprintf("Google Time: %v\n", time.Since(start)))
    
    return c.String(http.StatusOK, result.String())
})

	authGroup := e.Group("/auth")
	mountAuthRoutes(authGroup, h.Auth, e.IPExtractor, jwtIssuer)

	// Protected routes behind JWT auth
	protectedGroup := e.Group("")
	protectedGroup.Use(auth.RequireAuth(jwtIssuer))
	mountBettingRoutes(protectedGroup.Group("/v1/tickets"), h.Betting)

	return e
}

// mountAuthRoutes registers all /auth endpoints.
// The IPExtractor is passed through so clientIPKey uses the same
// proxy-aware IP resolution as the rest of the server.
func mountAuthRoutes(g *echo.Group, h *authhandler.AuthHandler, extractor echo.IPExtractor, jwtIssuer *auth.JWTIssuer) {
	loginRateLimit := echo.WrapMiddleware(
		httprate.LimitBy(5, time.Minute, clientIPKey(extractor)),
	)

	g.POST("/signup", h.Signup)
	g.POST("/verify", h.Verify)
	g.POST("/resend-otp", h.ResendOTP)
	g.POST("/login", h.Login, loginRateLimit)
	g.POST("/oauth/google", h.GoogleLogin, loginRateLimit)
	g.POST("/forgot-password", h.ForgotPassword)
	g.POST("/reset-password", h.ResetPassword)
	g.GET("/check-username", h.CheckUsername)
	g.POST("/refresh", h.Refresh)
	g.POST("/logout", h.Logout)

	// Protected auth routes — require a valid JWT.
	protected := g.Group("", auth.RequireAuth(jwtIssuer))
	protected.GET("/me", h.Me)
	protected.PATCH("/me", h.UpdateProfile)
}

// mountBettingRoutes registers all /v1/tickets endpoints.
// All routes here are behind the JWT auth middleware applied at the group level.
func mountBettingRoutes(g *echo.Group, h *bettinghandler.BettingHandler) {
	g.POST("/preview", h.Preview)
	g.POST("/track", h.Track)
	g.GET("", h.GetHistory)
	g.GET("/:id", h.GetTicketDetails)
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
