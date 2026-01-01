package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"vedio/worker/internal/config"

	"go.uber.org/zap"
)

// Client handles translation API calls to GLM.
type Client struct {
	apiKey string
	apiURL string
	client *http.Client
	logger *zap.Logger
}

// NewClient creates a new translation client.
func NewClient(cfg config.GLMConfig, logger *zap.Logger) *Client {
	return &Client{
		apiKey: cfg.APIKey,
		apiURL: cfg.APIURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Translate translates text from source language to target language.
func (c *Client) Translate(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
	// Create request body
	reqBody := map[string]interface{}{
		"source_language": sourceLang,
		"target_language": targetLang,
		"texts":          texts,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Make request with retry
	var resp *http.Response
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		resp, err = c.client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
			// Recreate request for retry
			req, _ = http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to call translation API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("translation API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp struct {
		Code    int      `json:"code"`
		Message string   `json:"message"`
		Data    []string `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("translation API error: %s", apiResp.Message)
	}

	if len(apiResp.Data) != len(texts) {
		return nil, fmt.Errorf("translation count mismatch: expected %d, got %d", len(texts), len(apiResp.Data))
	}

	return apiResp.Data, nil
}

