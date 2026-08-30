package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

// BrevoMailer sends emails via Brevo's REST API. Satisfies
// service.Mailer — see internal/auth/service/service.go.
type BrevoMailer struct {
	apiKey    string
	fromEmail string
}

// NewBrevoMailer builds a BrevoMailer.
func NewBrevoMailer(apiKey, fromEmail string) *BrevoMailer {
	return &BrevoMailer{
		apiKey:    apiKey,
		fromEmail: fromEmail,
	}
}

// brevoEmail represents the shape of an email in the Brevo API.
type brevoEmail struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// brevoPayload represents the JSON body sent to the Brevo API.
type brevoPayload struct {
	Sender      brevoEmail   `json:"sender"`
	To          []brevoEmail `json:"to"`
	Subject     string       `json:"subject"`
	HTMLContent string       `json:"htmlContent"`
}

// send is a private helper that handles the actual HTTP request to Brevo.
func (m *BrevoMailer) send(ctx context.Context, payload brevoPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("brevo marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.brevo.com/v3/smtp/email", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("brevo new request: %w", err)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("api-key", m.apiKey)
	req.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("brevo execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("brevo API error: status %d", resp.StatusCode)
	}

	return nil
}

// SendOTP emails a 6-digit OTP code to the given address.
func (m *BrevoMailer) SendOTP(ctx context.Context, toEmail, code string) error {
	if os.Getenv("LOG_OTP") == "true" {
		slog.Info("DEV MODE: OTP logged to console", "email", toEmail, "otp_code", code)
	}

	payload := brevoPayload{
		Sender:      brevoEmail{Email: m.fromEmail, Name: "SlipWise"},
		To:          []brevoEmail{{Email: toEmail}},
		Subject:     "Your SlipWise verification code",
		HTMLContent: brevoOtpEmailHTML(code), // Will reuse the clean HTML format
	}

	return m.send(ctx, payload)
}

// brevoOtpEmailHTML renders a modern, clean HTML body for the OTP email.
// Reuses the styling you already set up for Gmail for maximum compatibility.
func brevoOtpEmailHTML(code string) string {
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
func (m *BrevoMailer) SendFeedback(ctx context.Context, toEmail, userEmail, feedback string) error {
	payload := brevoPayload{
		Sender:  brevoEmail{Email: m.fromEmail, Name: "SlipWise System"},
		To:      []brevoEmail{{Email: toEmail}},
		Subject: fmt.Sprintf("SlipWise Feedback from %s", userEmail),
		HTMLContent: fmt.Sprintf(`
<div style="font-family: sans-serif; padding: 20px;">
  <h2>New Feedback via App</h2>
  <p><strong>From:</strong> %s</p>
  <hr />
  <p style="white-space: pre-wrap;">%s</p>
</div>`, userEmail, feedback),
	}

	return m.send(ctx, payload)
}
