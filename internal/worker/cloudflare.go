package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CloudflareClient defines the interface for communicating with the Cloudflare worker.
type CloudflareClient interface {
	FetchLiveMatches(ctx context.Context) ([]LiveMatch, error)
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
		httpClient: &http.Client{Timeout: 10 * time.Second},
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

	var matches []LiveMatch
	if err := json.NewDecoder(resp.Body).Decode(&matches); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return matches, nil
}
