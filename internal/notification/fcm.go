// Package notification provides the Firebase Cloud Messaging client.
package notification

import (
	"context"
	"log/slog"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
)

// Service defines the contract for sending push notifications.
type Service interface {
	SendMulticast(ctx context.Context, tokens []string, title, body string, data map[string]string) error
}

type fcmService struct {
	client *messaging.Client
}

// NewFCMService initializes the Firebase app using GOOGLE_APPLICATION_CREDENTIALS.
func NewFCMService(ctx context.Context) (Service, error) {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return nil, err
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	return &fcmService{client: client}, nil
}

func (s *fcmService) SendMulticast(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	if len(tokens) == 0 {
		return nil
	}

	// Firebase Multicast accepts up to 500 tokens per call.
	// We assume we won't exceed 500 tokens per ticket settlement right now,
	// but we could chunk the array here if needed.
	msg := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	br, err := s.client.SendEachForMulticast(ctx, msg)
	if err != nil {
		slog.Error("failed to send FCM multicast", "error", err)
		return err
	}

	if br.FailureCount > 0 {
		slog.Warn("fcm multicast had failures", "failure_count", br.FailureCount)
		// We could iterate over br.Responses to find which tokens failed
		// and purge them from our user_devices table.
	}

	return nil
}

// NoopService is used for local dev when Firebase isn't configured.
type NoopService struct{}

func (s *NoopService) SendMulticast(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	slog.Info("NOOP Notification Send", "title", title, "token_count", len(tokens))
	return nil
}
