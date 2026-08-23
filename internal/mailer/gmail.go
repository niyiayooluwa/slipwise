// Package mailer implements service.Mailer against Gmail's SMTP servers.
// Kept as its own package (rather than living in cmd/server)
// so it can be swapped for a fake in service-layer tests without
// touching main.go.
package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"os"
	"time"
)

// GmailMailer sends OTP emails via Gmail SMTP. Satisfies
// service.Mailer — see internal/auth/service/service.go.
type GmailMailer struct {
	email    string
	password string
	host     string
	port     string
}

// NewGmailMailer builds a GmailMailer. The password must be a 16-digit
// Google App Password, not your standard Gmail password.
func NewGmailMailer(email, appPassword string) *GmailMailer {
	return &GmailMailer{
		email:    email,
		password: appPassword,
		host:     "smtp.gmail.com",
		port:     "465",
	}
}

// sendMailTLS forces an IPv4 connection to the SMTP server using implicit TLS on port 465.
// This bypasses Railway dropping IPv6 connections to Gmail or blocking STARTTLS on port 587.
func sendMailTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp4", addr, &tls.Config{ServerName: "smtp.gmail.com"})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, "smtp.gmail.com")
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer c.Quit()

	if auth != nil {
		if err = c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err = c.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return fmt.Errorf("smtp rcpt: %w", err)
		}
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}

	return w.Close()
}

// SendOTP emails a 6-digit OTP code to the given address.
// Uses a goroutine to ensure request cancellation/timeouts from
// the calling HTTP handler actually propagate instead of a
// network request hanging past its deadline.
func (m *GmailMailer) SendOTP(ctx context.Context, toEmail, code string) error {
	if os.Getenv("LOG_OTP") == "true" {
		slog.Info("DEV MODE: OTP logged to console", "email", toEmail, "otp_code", code)
	}

	// Standard library requires us to build the MIME headers manually to send HTML
	headers := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	msg := fmt.Sprintf("To: %s\r\nSubject: Your SlipWise verification code\r\n%s\r\n%s",
		toEmail, headers, gmailOtpEmailHTML(code))

	auth := smtp.PlainAuth("", m.email, m.password, m.host)

	// Channel to catch the result of the SMTP call
	errChan := make(chan error, 1)

	go func() {
		errChan <- sendMailTLS(m.host+":"+m.port, auth, m.email, []string{toEmail}, []byte(msg))
	}()

	// Listen for either the context cancelling or the email sending
	select {
	case <-ctx.Done():
		return fmt.Errorf("gmail: send otp context cancelled/timeout: %w", ctx.Err())
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("gmail: send otp: %w", err)
		}
		return nil
	}
}

// gmailOtpEmailHTML renders a modern, clean HTML body for the OTP email.
// Uses inline styles and standard system fonts for maximum email client compatibility.
func gmailOtpEmailHTML(code string) string {
	return fmt.Sprintf(`
<div style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f9fafb; padding: 40px 20px; text-align: center;">
  <div style="max-width: 480px; margin: 0 auto; background-color: #ffffff; padding: 40px 30px; border-radius: 12px; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.05);">
    <div style="margin-bottom: 24px;">
      <!-- Email clients like Gmail aggressively strip out raw <svg> tags and data URIs for security reasons. -->
      <!-- To display the logo properly, you MUST host the logo as a PNG on a public server (like an S3 bucket or your Cloudflare worker) and update the src URL below. -->
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
func (m *GmailMailer) SendFeedback(ctx context.Context, toEmail, userEmail, feedback string) error {
	headers := "MIME-version: 1.0;\nContent-Type: text/plain; charset=\"UTF-8\";\n\n"
	msg := fmt.Sprintf("To: %s\r\nSubject: [SlipWise Feedback] from %s\r\n%s\r\nUser Email: %s\n\nFeedback:\n%s",
		toEmail, userEmail, headers, userEmail, feedback)

	auth := smtp.PlainAuth("", m.email, m.password, m.host)
	errChan := make(chan error, 1)

	go func() {
		errChan <- sendMailTLS(m.host+":"+m.port, auth, m.email, []string{toEmail}, []byte(msg))
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("gmail: send feedback context cancelled/timeout: %w", ctx.Err())
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("gmail: send feedback: %w", err)
		}
		return nil
	}
}
