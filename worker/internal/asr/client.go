package asr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"vedio/worker/internal/config"
	"vedio/worker/internal/models"

	"go.uber.org/zap"
)

// Client handles ASR API calls to the Moonshine ASR service.
type Client struct {
	baseURL string
	client  *http.Client
	logger  *zap.Logger
}

// NewClient creates a new ASR client.
func NewClient(cfg config.ASRConfig, logger *zap.Logger) *Client {
	return &Client{
		baseURL: cfg.URL,
		client: &http.Client{
			Timeout: 300 * time.Second, // ASR can take longer for long audio
		},
		logger: logger,
	}
}

// Recognize performs ASR using the Moonshine service.
// audioURL is the presigned MinIO URL accessible by the service.
func (c *Client) Recognize(ctx context.Context, audioURL string, language string) (*models.ASRResult, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("ASR_SERVICE_URL is required")
	}

	reqBody := map[string]interface{}{
		"audio_url": audioURL,
		"language":  language,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/transcribe", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call ASR service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ASR service returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Segments []struct {
			StartMs int    `json:"start_ms"`
			EndMs   int    `json:"end_ms"`
			Text    string `json:"text"`
		} `json:"segments"`
		Language   string `json:"language"`
		DurationMs int    `json:"duration_ms"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode ASR response: %w", err)
	}

	result := &models.ASRResult{
		Language:   apiResp.Language,
		DurationMs: apiResp.DurationMs,
		Segments:   make([]models.ASRSegment, 0, len(apiResp.Segments)),
	}
	for idx, seg := range apiResp.Segments {
		result.Segments = append(result.Segments, models.ASRSegment{
			Idx:     idx,
			StartMs: seg.StartMs,
			EndMs:   seg.EndMs,
			Text:    seg.Text,
		})
	}

	return result, nil
}
