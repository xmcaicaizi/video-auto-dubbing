package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"vedio/worker/internal/config"

	"go.uber.org/zap"
)

// Client handles TTS service API calls.
type Client struct {
	baseURL string
	client  *http.Client
	logger  *zap.Logger
}

// SynthesisRequest represents a TTS synthesis request.
type SynthesisRequest struct {
    Text            string                 `json:"text"`
    SpeakerID       string                 `json:"speaker_id"`
    PromptAudioURL  string                 `json:"prompt_audio_url,omitempty"`
    TargetDurationMs int                   `json:"target_duration_ms"`
    Language        string                 `json:"language"`
    ProsodyControl  map[string]interface{} `json:"prosody_control,omitempty"`
    OutputFormat    string                 `json:"output_format"`
    SampleRate      int                    `json:"sample_rate"`
}

// SynthesisResponse represents a TTS synthesis response.
type SynthesisResponse struct {
	AudioURL    string `json:"audio_url"`
	DurationMs int    `json:"duration_ms"`
	SampleRate  int    `json:"sample_rate"`
	Format      string `json:"format"`
	FileSize    int    `json:"file_size"`
}

// NewClient creates a new TTS client.
func NewClient(cfg config.TTSConfig, logger *zap.Logger) *Client {
	return &Client{
		baseURL: cfg.URL,
		client: &http.Client{
			Timeout: 600 * time.Second, // TTS can take longer for IndexTTS2
		},
		logger: logger,
	}
}

// Synthesize performs TTS synthesis.
func (c *Client) Synthesize(ctx context.Context, req SynthesisRequest, modelScopeToken string) (io.ReadCloser, error) {
	// Create request body
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/synthesize", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if modelScopeToken != "" {
		httpReq.Header.Set("X-ModelScope-Token", modelScopeToken)
	}

	// Make request with retry
	var resp *http.Response
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		resp, err = c.client.Do(httpReq)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
			// Recreate request for retry
			httpReq, _ = http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Accept", "application/json")
			if modelScopeToken != "" {
				httpReq.Header.Set("X-ModelScope-Token", modelScopeToken)
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to call TTS API: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("TTS API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp SynthesisResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Download audio from URL
	if apiResp.AudioURL != "" {
		audioURL := apiResp.AudioURL
		if strings.HasPrefix(audioURL, "/") {
			audioURL = strings.TrimRight(c.baseURL, "/") + audioURL
		}
		audioReq, err := http.NewRequestWithContext(ctx, "GET", audioURL, nil)
		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to create audio request: %w", err)
		}

		audioResp, err := c.client.Do(audioReq)
		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to download audio: %w", err)
		}

		if audioResp.StatusCode != http.StatusOK {
			audioResp.Body.Close()
			resp.Body.Close()
			return nil, fmt.Errorf("failed to download audio: status %d", audioResp.StatusCode)
		}

		resp.Body.Close()
		return audioResp.Body, nil
	}

	// If no URL, assume response body contains audio (for future direct audio response)
	return resp.Body, nil
}

