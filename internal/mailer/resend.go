// Package mailer implements service.Mailer against the real Resend
// API. Kept as its own package (rather than living in cmd/server)
// so it can be swapped for a fake in service-layer tests without
// touching main.go.
package mailer

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v3"
)

// ResendMailer sends OTP emails via Resend. Satisfies
// service.Mailer — see internal/auth/service/service.go.
type ResendMailer struct {
	client *resend.Client
	from   string
}

// NewResendMailer builds a ResendMailer. from must be an address on a
// domain verified in your Resend dashboard (e.g.
// "Sportloga <otp@sportloga.app>") — sends will fail otherwise.
func NewResendMailer(apiKey, from string) *ResendMailer {
	return &ResendMailer{
		client: resend.NewClient(apiKey),
		from:   from,
	}
}

// SendOTP emails a 6-digit OTP code to the given address. Uses
// SendWithContext (not Send) so request cancellation/timeouts from
// the calling HTTP handler actually propagate to the Resend call
// instead of a signup request hanging past its deadline.
func (m *ResendMailer) SendOTP(ctx context.Context, email, code string) error {
	params := &resend.SendEmailRequest{
		From:    m.from,
		To:      []string{email},
		Subject: "Your Sportloga verification code",
		Html:    otpEmailHTML(code),
	}

	_, err := m.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("resend: send otp: %w", err)
	}
	return nil
}

// otpEmailHTML renders a minimal HTML body for the OTP email. Kept
// deliberately plain (no external CSS/images) — OTP emails need to
// render correctly in every client, not look polished.
func otpEmailHTML(code string) string {
	return fmt.Sprintf(`
<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">
  <p>Your Sportloga verification code is:</p>
  <p style="font-size: 32px; font-weight: bold; letter-spacing: 4px;">%s</p>
  <p style="color: #666;">This code expires in 10 minutes. If you didn't request this, you can ignore this email.</p>
</div>`, code)
}
