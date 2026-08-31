// Package worker handles asynchronous background tasks, scheduled polling,
// and third-party integrations (Cloudflare scraper proxy, FCM push notifications).
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type cachedResponse struct {
	data      []byte
	expiresAt time.Time
}

// In-Memory RAM Cache:
// Prevents "Thundering Herd" API rate limits.
// When 100 users preview or track the same viral betslip code within 60 seconds,
// the server fetches it from SportyBet once, caches the raw JSON, and serves the rest instantly from RAM.
var (
	ticketCache sync.Map
	cacheTTL    = 60 * time.Second
)

// CloudflareClient defines the transport interface for communicating with our Cloudflare Worker proxy.
// We proxy SportyBet traffic through Cloudflare Workers to bypass geo-restrictions and basic IP blocks.
type CloudflareClient interface {
	FetchLiveMatches(ctx context.Context) ([]LiveMatch, error)
	FetchTicketByCode(ctx context.Context, shareCode string) ([]byte, error)
}

// LiveMatch represents a live match fetched from the firehose.
type LiveMatch struct {
	ProviderID string          `json:"provider_id"`
	Data       json.RawMessage `json:"data"` // Raw data from the provider
}

type cloudflareClientImpl struct {
	httpClient *http.Client
	workerURL  string
}

// NewCloudflareClient creates a new CloudflareClient.
func NewCloudflareClient(workerURL string) CloudflareClient {
	return &cloudflareClientImpl{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		workerURL:  workerURL,
	}
}

func (c *cloudflareClientImpl) FetchLiveMatches(ctx context.Context) ([]LiveMatch, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.workerURL+"/live", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

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
				Status  int             `json:"status"` // 1=live, 2=ended, etc
				Raw     json.RawMessage `json:"-"`
			} `json:"events"`
		} `json:"data"`
	}

	// We need the raw json for each event, so decode differently
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read live matches body: %w", err)
	}

	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode live matches: %w", err)
	}

	// But we also need the raw json of the event.
	// Since we unmarshaled the whole thing, we lost the raw bytes of each event.
	// Let's do a trick using map[string]interface{} or just rawmessage
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

func (c *cloudflareClientImpl) FetchTicketByCode(ctx context.Context, shareCode string) ([]byte, error) {
	// 1. Check RAM Cache
	if val, ok := ticketCache.Load(shareCode); ok {
		cached := val.(cachedResponse)
		if time.Now().Before(cached.expiresAt) {
			return cached.data, nil
		}
		ticketCache.Delete(shareCode) // Evict expired
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.workerURL+"/ticket?code="+shareCode, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ticket by code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Readall the body since it's just raw json to be unmarshaled by the translator
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 3. Save to RAM Cache
	ticketCache.Store(shareCode, cachedResponse{
		data:      data,
		expiresAt: time.Now().Add(cacheTTL),
	})

	return data, nil
}
