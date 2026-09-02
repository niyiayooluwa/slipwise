// Package mailer implements service.Mailer against the real Resend
// API. Kept as its own package (rather than living in cmd/server)
// so it can be swapped for a fake in service-layer tests without
// touching main.go.
package mailer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/resend/resend-go/v3"
)

// ResendMailer sends OTP emails via Resend. Satisfies
// service.Mailer — see internal/auth/service/service.go.
type ResendMailer struct {
	client      *resend.Client
	domain      string
	defaultFrom string
}

// NewResendMailer builds a ResendMailer. 'from' can be either a full email address
// (e.g. "SlipWise <noreply@mail.slipwise.niyiayo.com>") or just the domain ("mail.slipwise.niyiayo.com").
// Specialized prefixes (auth@, feedback@, support@) are automatically derived.
func NewResendMailer(apiKey, from string) *ResendMailer {
	domain := strings.TrimSpace(from)
	if idx := strings.Index(domain, "@"); idx != -1 {
		domain = strings.TrimSuffix(domain[idx+1:], ">")
		domain = strings.TrimSpace(domain)
	}

	return &ResendMailer{
		client:      resend.NewClient(apiKey),
		domain:      domain,
		defaultFrom: from,
	}
}

// fromAddress constructs a branded sender address using the configured domain.
func (m *ResendMailer) fromAddress(name, prefix string) string {
	if m.domain != "" {
		return fmt.Sprintf("%s <%s@%s>", name, prefix, m.domain)
	}
	return m.defaultFrom
}

// SendOTP emails a 6-digit OTP code to the given address from auth@<domain>.
func (m *ResendMailer) SendOTP(ctx context.Context, email, code string) error {
	if os.Getenv("LOG_OTP") == "true" {
		slog.Info("DEV MODE: OTP logged to console", "email", email, "otp_code", code)
	}

	if os.Getenv("MOCK_EMAIL") == "true" {
		slog.Info("MOCK_EMAIL intercept", "email", email, "otp_code", code)
		return nil
	}

	params := &resend.SendEmailRequest{
		From:    m.fromAddress("SlipWise Security", "auth"),
		To:      []string{email},
		Subject: "Your SlipWise verification code",
		Html:    resendOtpEmailHTML(code),
	}

	_, err := m.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("resend: send otp: %w", err)
	}
	return nil
}

// resendOtpEmailHTML renders a modern, clean HTML body for the OTP email.
func resendOtpEmailHTML(code string) string {
	return fmt.Sprintf(`
<div style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f9fafb; padding: 40px 20px; text-align: center;">
  <div style="max-width: 480px; margin: 0 auto; background-color: #ffffff; padding: 40px 30px; border-radius: 12px; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.05);">
    <div style="margin-bottom: 24px;">
      <img src="https://raw.githubusercontent.com/niyiayooluwa/assets/main/orange.png" alt="SlipWise Logo" width="48" height="48" style="display: block; margin: 0 auto; border-radius: 8px;" />
    </div>
    <h2 style="margin-top: 0; color: #111827; font-size: 24px;">Welcome to SlipWise</h2>
    <p style="color: #4b5563; font-size: 16px; line-height: 1.5; margin-bottom: 30px;">
      Use the following verification code to complete your setup.
    </p>
    <div style="background-color: #f3f4f6; border-radius: 8px; padding: 12px; margin-bottom: 30px;">
      <p style="font-size: 24px; font-weight: 800; letter-spacing: 6px; color: #111827; margin: 0;">%s</p>
    </div>
    <p style="color: #6b7280; font-size: 14px; margin: 0;">
      This code expires in 10 minutes. If you didn't request this, you can safely ignore this email.
    </p>
  </div>
</div>`, code)
}

// SendFeedback emails user feedback to the admin.
func (m *ResendMailer) SendFeedback(ctx context.Context, toEmail, userEmail, feedback string) error {
	if os.Getenv("MOCK_EMAIL") == "true" {
		slog.Info("MOCK_EMAIL intercept feedback", "to", toEmail, "fromUser", userEmail, "feedback", feedback)
		return nil
	}

	htmlBody := fmt.Sprintf(`
<div style="font-family: sans-serif; padding: 20px;">
  <h2>New Feedback via App</h2>
  <p><strong>From:</strong> %s</p>
  <hr />
  <p style="white-space: pre-wrap;">%s</p>
</div>`, userEmail, feedback)

	params := &resend.SendEmailRequest{
		From:    m.fromAddress("SlipWise Feedback", "feedback"),
		To:      []string{toEmail},
		Subject: fmt.Sprintf("SlipWise Feedback from %s", userEmail),
		Html:    htmlBody,
	}

	_, err := m.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("resend: send feedback: %w", err)
	}
	return nil
}
