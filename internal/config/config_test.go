package config_test

import (
	"strings"
	"testing"

	"sportloga/internal/config"
)

// t.Setenv sets an env var for just this test and automatically
// restores whatever it was before once the test finishes — safer than
// os.Setenv, which would leak into other tests running afterward.

func TestLoad_Success(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("RESEND_API_KEY", "re_test")
	t.Setenv("RESEND_FROM_ADDRESS", "Test <test@example.com>")
	t.Setenv("GOOGLE_CLIENT_ID", "test")
	t.Setenv("CLOUDFLARE_WORKER_URL", "https://worker.dev")
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
	t.Setenv("RESEND_API_KEY", "")
	t.Setenv("RESEND_FROM_ADDRESS", "")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("CLOUDFLARE_WORKER_URL", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected an error when required vars are missing, got none")
	}

	// Check the error actually names what's missing, not just that
	// something failed — this is what makes Load's error useful to a
	// person reading it, and worth locking in with a test.
	for _, want := range []string{"DATABASE_URL", "JWT_SECRET", "RESEND_API_KEY", "RESEND_FROM_ADDRESS", "GOOGLE_CLIENT_ID", "CLOUDFLARE_WORKER_URL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to mention %s, got: %v", want, err)
		}
	}
}

func TestLoad_CustomPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("RESEND_API_KEY", "re_test")
	t.Setenv("RESEND_FROM_ADDRESS", "Test <test@example.com>")
	t.Setenv("GOOGLE_CLIENT_ID", "test")
	t.Setenv("CLOUDFLARE_WORKER_URL", "https://worker.dev")
	t.Setenv("PORT", "9000")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Port != "9000" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9000")
	}
}
