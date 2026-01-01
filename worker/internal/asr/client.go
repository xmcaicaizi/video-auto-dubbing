package asr

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"vedio/worker/internal/config"
	"vedio/worker/internal/models"

	"go.uber.org/zap"
)

// Client handles ASR API calls to VolcEngine.
type Client struct {
	accessKey string
	secretKey string
	baseURL   string
	client    *http.Client
	logger    *zap.Logger
}

// NewClient creates a new ASR client.
func NewClient(cfg config.VolcEngineASRConfig, logger *zap.Logger) *Client {
	return &Client{
		accessKey: cfg.AccessKey,
		secretKey: cfg.SecretKey,
		baseURL:   "https://open.volcengineapi.com",
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}
}

// Recognize performs ASR on an audio file.
func (c *Client) Recognize(ctx context.Context, audioReader io.Reader, language string) (*models.ASRResult, error) {
	// Read audio data
	audioData, err := io.ReadAll(audioReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio: %w", err)
	}

	// Create request body
	reqBody := map[string]interface{}{
		"app_id": "default",
		"language": language,
		"audio_format": "wav",
		"sample_rate": 22050,
		"audio_data": hex.EncodeToString(audioData),
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/asr", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.generateAuthHeader(req))

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
			req, _ = http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", c.generateAuthHeader(req))
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to call ASR API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ASR API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Segments []struct {
				StartTime float64 `json:"start_time"`
				EndTime   float64 `json:"end_time"`
				Text      string  `json:"text"`
			} `json:"segments"`
			Duration float64 `json:"duration"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("ASR API error: %s", apiResp.Message)
	}

	// Convert to our format
	result := &models.ASRResult{
		Language:   language,
		DurationMs: int(apiResp.Data.Duration * 1000),
		Segments:   make([]models.ASRSegment, 0, len(apiResp.Data.Segments)),
	}

	for idx, seg := range apiResp.Data.Segments {
		result.Segments = append(result.Segments, models.ASRSegment{
			Idx:     idx,
			StartMs: int(seg.StartTime * 1000),
			EndMs:   int(seg.EndTime * 1000),
			Text:    seg.Text,
		})
	}

	return result, nil
}

// generateAuthHeader generates authorization header for VolcEngine API.
func (c *Client) generateAuthHeader(req *http.Request) string {
	// Simplified auth - in production, use proper VolcEngine signature algorithm
	timestamp := time.Now().Unix()
	nonce := fmt.Sprintf("%d", timestamp)
	
	// Create signature
	signStr := fmt.Sprintf("%s%s%s", c.accessKey, nonce, c.secretKey)
	hash := sha256.Sum256([]byte(signStr))
	signature := hex.EncodeToString(hash[:])

	return fmt.Sprintf("Bearer %s:%s:%s", c.accessKey, nonce, signature)
}

