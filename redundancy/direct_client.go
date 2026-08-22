package redundancy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LiveMatch represents a live match fetched from the firehose.
type LiveMatch struct {
	ProviderID string          `json:"provider_id"`
	Data       json.RawMessage `json:"data"`
}

// DirectClient defines the interface for communicating directly with Sportybet.
type DirectClient interface {
	FetchLiveMatches(ctx context.Context) ([]LiveMatch, error)
	FetchTicketByCode(ctx context.Context, shareCode string) ([]byte, error)
}

type directClientImpl struct {
	httpClient *http.Client
}

// NewDirectClient creates a client that bypasses Cloudflare workers and hits Sportybet directly.
func NewDirectClient() DirectClient {
	return &directClientImpl{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *directClientImpl) FetchLiveMatches(ctx context.Context) ([]LiveMatch, error) {
	url := "https://www.sportybet.com/api/ng/factsCenter/configurableLiveOrPrematchEvents?sportId=sr:sport:1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Masquerade as a standard browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://www.sportybet.com")
	req.Header.Set("Referer", "https://www.sportybet.com/ng/")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch live matches: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var payload struct {
		Data []struct {
			Events []struct {
				EventID string          `json:"eventId"`
				Status  int             `json:"status"`
				Raw     json.RawMessage `json:"-"`
			} `json:"events"`
		} `json:"data"`
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read live matches body: %w", err)
	}

	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode live matches: %w", err)
	}

	var rawPayload struct {
		Data []struct {
			Events []json.RawMessage `json:"events"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &rawPayload); err != nil {
		return nil, err
	}

	var matches []LiveMatch
	for i, category := range rawPayload.Data {
		for j, rawEvent := range category.Events {
			eventID := payload.Data[i].Events[j].EventID
			matches = append(matches, LiveMatch{
				ProviderID: eventID,
				Data:       rawEvent,
			})
		}
	}

	return matches, nil
}

func (c *directClientImpl) FetchTicketByCode(ctx context.Context, shareCode string) ([]byte, error) {
	// Cache bust with timestamp
	url := fmt.Sprintf("https://www.sportybet.com/api/ng/orders/share/%s?_t=%d", shareCode, time.Now().UnixMilli())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://www.sportybet.com")
	req.Header.Set("Referer", "https://www.sportybet.com/ng/")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ticket by code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return data, nil
}
