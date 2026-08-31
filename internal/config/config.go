// Package config centralizes all of the application's startup configuration.
//
// Architectural Philosophy:
// Rather than scattering `os.Getenv(...)` calls across handlers, services, and workers
// (which causes silent bugs when an env var is missing 3 layers deep at runtime),
// we parse and validate EVERYTHING up front during application boot.
//
// If a variable is missing, the application halts immediately with a clear, sorted list
// of all missing variables so the developer doesn't have to restart 10 times to fix 10 errors.
package config

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds every environment-derived value the app needs at startup.
// Rule of Thumb: If a component needs a configuration value, add it here and pass it down.
// Never call `os.Getenv` inside business logic packages.
type Config struct {
	// DatabaseURL is the Postgres connection string, e.g.
	// postgres://user:pass@host:5432/dbname?sslmode=disable
	DatabaseURL string
	// JWTSecret signs and verifies access tokens. Must be a real
	// generated secret in staging/prod, not a dev placeholder.
	JWTSecret string
	// BrevoAPIKey is the API key used to authenticate with Brevo.
	BrevoAPIKey string
	// BrevoSenderEmail is the authorized sender email address in Brevo.
	BrevoSenderEmail string
	// Port is the HTTP port the server listens on. Optional — defaults
	// to "8080" if unset, since a missing PORT shouldn't be fatal the
	// way a missing DATABASE_URL is.
	Port string
	// AllowedOrigins lists the frontend origins allowed to call this
	// API cross-origin (e.g. "https://app.slipwise.com"). Optional —
	// defaults to ["*"] for local dev, where nothing outside your own
	// machine needs to be locked down yet. Set CORS_ALLOWED_ORIGINS as
	// a comma-separated list once there's a real frontend origin to
	// restrict to.
	AllowedOrigins []string
	// TrustedProxyCIDRs is the list of CIDR ranges representing trusted
	// reverse proxies (e.g. "10.0.0.0/8,172.16.0.0/12"). When empty the
	// server is assumed to be directly on the internet and r.RemoteAddr is
	// used as the client IP. When set, X-Forwarded-For is read and the
	// header is trusted only from IPs that fall inside one of these ranges.
	// Changing deployment topology never requires a code change — only an
	// update to this env var.
	TrustedProxyCIDRs []string
	// GoogleClientID is the OAuth client ID for the Flutter app.
	// Used to verify ID tokens from Google Sign-In.
	GoogleClientID string
	// CloudflareWorkerURL is the URL of the Cloudflare worker proxy.
	CloudflareWorkerURL string
	// FeedbackEmail is the admin email address where user feedback is sent.
	FeedbackEmail string
	// FirebaseCredentialsJSON is the raw JSON key for Firebase Admin SDK.
	FirebaseCredentialsJSON string
	// CronSecret is an optional shared secret to protect /internal/cron endpoints.
	// If set, requests must include the header: X-Cron-Secret: <value>.
	// If unset, the cron endpoint is open (not recommended for production).
	CronSecret string
}

// Load reads .env (if present) into the process environment, then
// reads and validates every required variable into a Config. Returns
// an error listing every missing variable at once — not just the
// first one encountered — so a person fixing their .env doesn't have
// to run the binary repeatedly to discover each missing var one at a
// time.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from process environment")
	}

	cfg := &Config{
		DatabaseURL:             os.Getenv("DATABASE_URL"),
		JWTSecret:               os.Getenv("JWT_SECRET"),
		BrevoAPIKey:             os.Getenv("BREVO_API_KEY"),
		BrevoSenderEmail:        os.Getenv("BREVO_SENDER_EMAIL"),
		Port:                    os.Getenv("PORT"),
		GoogleClientID:          os.Getenv("GOOGLE_CLIENT_ID"),
		CloudflareWorkerURL:     os.Getenv("CLOUDFLARE_WORKER_URL"),
		FeedbackEmail:           os.Getenv("FEEDBACK_EMAIL"),
		FirebaseCredentialsJSON: os.Getenv("FIREBASE_CREDENTIALS_JSON"),
		CronSecret:              os.Getenv("CRON_SECRET"), // Optional — empty = open endpoint
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" {
		cfg.AllowedOrigins = strings.Split(origins, ",")
	} else {
		cfg.AllowedOrigins = []string{"*"}
	}

	if cidrs := os.Getenv("TRUSTED_PROXY_CIDRS"); cidrs != "" {
		cfg.TrustedProxyCIDRs = strings.Split(cidrs, ",")
	}

	required := map[string]string{
		"DATABASE_URL":              cfg.DatabaseURL,
		"JWT_SECRET":                cfg.JWTSecret,
		"BREVO_API_KEY":             cfg.BrevoAPIKey,
		"BREVO_SENDER_EMAIL":        cfg.BrevoSenderEmail,
		"GOOGLE_CLIENT_ID":          cfg.GoogleClientID,
		"CLOUDFLARE_WORKER_URL":     cfg.CloudflareWorkerURL,
		"FEEDBACK_EMAIL":            cfg.FeedbackEmail,
		"FIREBASE_CREDENTIALS_JSON": cfg.FirebaseCredentialsJSON,
	}

	var missing []string
	for name, val := range required {
		if val == "" {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing) // deterministic order, not Go map iteration order
		return nil, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}
