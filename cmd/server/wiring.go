package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// mustGetEnv reads a required environment variable or exits — fine
// for a handful of startup secrets, swap for internal/config once
// there are more than a couple of these.
func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var: %s", key)
	}
	return v
}

// mustConnectDB opens the pgx pool used by every domain's repository
// layer. Panics on failure since the server is useless without a DB.
func mustConnectDB() *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), mustGetEnv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	return pool
}

// resendMailer is a placeholder implementing service.Mailer over the
// Resend API. Wire in the real SDK call here — kept minimal since it's
// outside today's scope (auth layering).
type resendMailer struct {
	apiKey string
}

func newResendMailer(apiKey string) *resendMailer {
	return &resendMailer{apiKey: apiKey}
}

func (m *resendMailer) SendOTP(ctx context.Context, email, code string) error {
	// TODO: call Resend's send-email endpoint with the OTP code.
	log.Printf("TODO: send OTP %s to %s via Resend", code, email)
	return nil
}
