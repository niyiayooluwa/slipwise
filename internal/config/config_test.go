package config_test

import (
	"strings"
	"testing"

	"slipwise/internal/config"
)

// t.Setenv sets an env var for just this test and automatically
// restores whatever it was before once the test finishes — safer than
// os.Setenv, which would leak into other tests running afterward.

func TestLoad_Success(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("BREVO_API_KEY", "test-key")
	t.Setenv("BREVO_SENDER_EMAIL", "test@test.com")
	t.Setenv("GOOGLE_CLIENT_ID", "test")
	t.Setenv("CLOUDFLARE_WORKER_URL", "https://worker.dev")
	t.Setenv("FEEDBACK_EMAIL", "test@admin.com")
	t.Setenv("FIREBASE_CREDENTIALS_JSON", "{}")
	// PORT deliberately left unset to also exercise the default below.

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DatabaseURL != "postgres://localhost/test" {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgres://localhost/test")
	}
	if cfg.Port != "8080" {
		t.Errorf("expected default Port 8080 when PORT unset, got %q", cfg.Port)
	}
}

func TestLoad_MissingVars(t *testing.T) {
	// Explicitly blank everything, in case a previous test (or the
	// real environment this runs in) left something set.
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("BREVO_API_KEY", "")
	t.Setenv("BREVO_SENDER_EMAIL", "")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("CLOUDFLARE_WORKER_URL", "")
	t.Setenv("FEEDBACK_EMAIL", "")
	t.Setenv("FIREBASE_CREDENTIALS_JSON", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected an error when required vars are missing, got none")
	}

	// Check the error actually names what's missing, not just that
	// something failed — this is what makes Load's error useful to a
	// person reading it, and worth locking in with a test.
	for _, want := range []string{"DATABASE_URL", "JWT_SECRET", "BREVO_API_KEY", "BREVO_SENDER_EMAIL", "GOOGLE_CLIENT_ID", "CLOUDFLARE_WORKER_URL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to mention %s, got: %v", want, err)
		}
	}
}

func TestLoad_CustomPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("BREVO_API_KEY", "test-key")
	t.Setenv("BREVO_SENDER_EMAIL", "test@test.com")
	t.Setenv("GOOGLE_CLIENT_ID", "test")
	t.Setenv("CLOUDFLARE_WORKER_URL", "https://worker.dev")
	t.Setenv("FEEDBACK_EMAIL", "test@admin.com")
	t.Setenv("FIREBASE_CREDENTIALS_JSON", "{}")
	t.Setenv("PORT", "9000")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Port != "9000" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9000")
	}
}
