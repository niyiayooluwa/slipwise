// Package config centralizes all of the app's startup configuration
// in one place. Before this existed, main.go called mustGetEnv
// scattered across itself — which meant a missing env var only
// surfaced whenever that particular line happened to run, sometimes
// deep into startup after a DB connection was already open. Load
// validates everything up front, once, so the app either starts
// clean or fails immediately with a full list of what's missing.
package config

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds every environment-derived value the app needs at
// startup. Add a field here (and to Load's required-vars check, if
// it's not optional) rather than reading os.Getenv directly anywhere
// else in the codebase — that's the whole point of centralizing this.
type Config struct {
	// DatabaseURL is the Postgres connection string, e.g.
	// postgres://user:pass@host:5432/dbname?sslmode=disable
	DatabaseURL string
	// JWTSecret signs and verifies access tokens. Must be a real
	// generated secret in staging/prod, not a dev placeholder.
	JWTSecret string
	// GmailUser is the Gmail address used for sending OTPs.
	GmailUser string
	// GmailAppPassword is a 16-character App Password, NOT the standard account password.
	GmailAppPassword string
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
		GmailUser:               os.Getenv("GMAIL_USER"),
		GmailAppPassword:        os.Getenv("GMAIL_APP_PASSWORD"),
		Port:                    os.Getenv("PORT"),
		GoogleClientID:          os.Getenv("GOOGLE_CLIENT_ID"),
		CloudflareWorkerURL:     os.Getenv("CLOUDFLARE_WORKER_URL"),
		FeedbackEmail:           os.Getenv("FEEDBACK_EMAIL"),
		FirebaseCredentialsJSON: os.Getenv("FIREBASE_CREDENTIALS_JSON"),
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
		"GMAIL_USER":                cfg.GmailUser,
		"GMAIL_APP_PASSWORD":        cfg.GmailAppPassword,
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
