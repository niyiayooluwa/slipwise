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
// "slipwise <otp@slipwise.app>") — sends will fail otherwise.
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
	if os.Getenv("MOCK_EMAIL") == "true" {
		slog.Info("MOCK_EMAIL intercept", "email", email, "otp_code", code)
		return nil
	}

	params := &resend.SendEmailRequest{
		From:    m.from,
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

// resendOtpEmailHTML renders a minimal HTML body for the OTP email. Kept
// deliberately plain (no external CSS/images) — OTP emails need to
// render correctly in every client, not look polished.
func resendOtpEmailHTML(code string) string {
	return fmt.Sprintf(`
<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto; text-align: center;">
  <div style="margin-bottom: 24px;">
    <svg width="48" height="48" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path fill-rule="evenodd" clip-rule="evenodd" d="M18.3279 33.2066C19.157 32.2118 20.3851 31.6365 21.6802 31.6365H39.2727C44.0927 31.6365 48 27.7292 48 22.9093V18.5456C48 13.7257 44.0927 9.81836 39.2727 9.81836H35.862C34.567 9.81836 33.3388 10.3936 32.5098 11.3885L29.6721 14.7937C28.843 15.7886 27.6149 16.3638 26.3198 16.3638H8.72727C3.90733 16.3638 0 20.2711 0 25.0911V29.4547C0 34.2747 3.90733 38.182 8.72727 38.182H12.138C13.433 38.182 14.6612 37.6068 15.4902 36.6119L18.3279 33.2066ZM41.4545 18.5456C41.4545 17.3406 40.4777 16.3638 39.2727 16.3638H32.5893C31.2942 16.3638 30.0661 16.939 29.237 17.9339L26.3993 21.3392C25.5703 22.334 24.3421 22.9093 23.0471 22.9093H8.72727C7.52229 22.9093 6.54545 23.8861 6.54545 25.0911V29.4547C6.54545 30.6597 7.52229 31.6365 8.72727 31.6365H15.4107C16.7058 31.6365 17.9339 31.0613 18.763 30.0664L21.6007 26.6612C22.4297 25.6663 23.6579 25.0911 24.9529 25.0911H39.2727C40.4777 25.0911 41.4545 24.1143 41.4545 22.9093V18.5456Z" fill="#BC3B00"/>
    </svg>
  </div>
  <p>Your SlipWise verification code is:</p>
  <p style="font-size: 32px; font-weight: bold; letter-spacing: 4px;">%s</p>
  <p style="color: #666;">This code expires in 10 minutes. If you didn't request this, you can ignore this email.</p>
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
		From:    m.from,
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
