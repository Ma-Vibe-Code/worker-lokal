package client

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Ma-Vibe-Code/worker-lokal/internal/config"
	"github.com/Ma-Vibe-Code/worker-lokal/internal/models"
)

// APIClient handles communication with the central management backend REST API.
type APIClient struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewAPIClient creates an instance of APIClient with configured timeouts.
func NewAPIClient(cfg *config.Config) *APIClient {
	return &APIClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// FetchCameras fetches the initial or updated list of assigned cameras for this worker.
func (c *APIClient) FetchCameras() ([]models.Camera, error) {
	req, err := http.NewRequest(http.MethodGet, c.cfg.APIBaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Worker-ID", c.cfg.WorkerID)

	if c.cfg.APIAuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.cfg.APIAuthToken))
	}

	log.Printf("[API] Fetching camera configurations from %s (Worker: %s)...", c.cfg.APIBaseURL, c.cfg.WorkerID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned non-200 status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp models.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	log.Printf("[API] Successfully fetched %d camera(s)", len(apiResp.Data))
	return apiResp.Data, nil
}
