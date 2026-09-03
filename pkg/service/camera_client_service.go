package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2/log"

	"github.com/Ma-Vibe-Code/worker-lokal/config"
	"github.com/Ma-Vibe-Code/worker-lokal/pkg/dto"
	"github.com/Ma-Vibe-Code/worker-lokal/pkg/model"
)

// CameraClientService handles communication with the central management backend REST API.
type CameraClientService struct {
	httpClient *http.Client
}

// NewCameraClientService creates an instance of CameraClientService with standard timeouts.
func NewCameraClientService() *CameraClientService {
	return &CameraClientService{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// FetchCameras fetches the list of assigned cameras for this worker from the central API.
func (c *CameraClientService) FetchCameras() ([]model.Camera, error) {
	apiBaseURL := config.API_BASE_URL.GetValue()
	workerID := config.WORKER_ID.GetValue()
	apiKeyHeader := config.API_KEY_HEADER.GetValueOrDefault("x-api-key")
	apiKey := config.API_KEY.GetValue()
	apiAuthToken := config.API_AUTH_TOKEN.GetValue()

	if apiBaseURL == "" {
		return nil, fmt.Errorf("API_BASE_URL is not configured")
	}

	req, err := http.NewRequest(http.MethodGet, apiBaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if workerID != "" {
		req.Header.Set("X-Worker-ID", workerID)
	}

	if apiKey != "" {
		req.Header.Set(apiKeyHeader, apiKey)
	}

	if apiAuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiAuthToken))
	}

	log.Infof("[API] Fetching camera configurations from %s (Worker: %s)...", apiBaseURL, workerID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned non-200 status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp dto.APIResponseDTO
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	log.Infof("[API] Successfully fetched %d camera(s)", len(apiResp.Data))
	return apiResp.Data, nil
}
