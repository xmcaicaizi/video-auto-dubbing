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

	"vedio/shared/config"

	"go.uber.org/zap"
)

// Client defines the interface for TTS services.
type Client interface {
	Synthesize(ctx context.Context, req SynthesisRequest) (io.ReadCloser, error)
}

// SynthesisRequest represents a TTS synthesis request.
type SynthesisRequest struct {
	Text              string                 `json:"text"`
	SpeakerID         string                 `json:"speaker_id"`
	PromptAudioURL    string                 `json:"prompt_audio_url,omitempty"`
	TargetDurationMs  int                    `json:"target_duration_ms"`
	Language          string                 `json:"language"`
	ProsodyControl    map[string]interface{} `json:"prosody_control,omitempty"`
	OutputFormat      string                 `json:"output_format"`
	SampleRate        int                    `json:"sample_rate"`
	TTSBackend        string                 `json:"tts_backend,omitempty"`
	IndexTTSGradioURL string                 `json:"indextts_gradio_url,omitempty"`
}

// SynthesisResponse represents a TTS synthesis response.
type SynthesisResponse struct {
	AudioURL   string `json:"audio_url"`
	DurationMs int    `json:"duration_ms"`
	SampleRate int    `json:"sample_rate"`
	Format     string `json:"format"`
	FileSize   int    `json:"file_size"`
}

// NewClient creates the appropriate TTS client based on configuration.
// It selects between:
// - GradioClient: for Gradio-based IndexTTS services (explicit backend="gradio")
// - VLLMClient: for index-tts-vllm remote service (default, supports /tts_url)
// - LegacyClient: for backward compatibility with old tts_service
func NewClient(cfg config.TTSConfig, logger *zap.Logger) Client {
	// Use legacy client for backward compatibility
	if cfg.Backend == "legacy" || cfg.Backend == "local" {
		return NewLegacyClient(cfg, logger)
	}

	// Explicit Gradio backend selection
	if cfg.Backend == "gradio" {
		logger.Info("Using GradioClient (explicit backend=gradio)",
			zap.String("url", cfg.URL))
		return NewGradioClient(cfg, logger)
	}

	// Auto-detect Gradio interface by checking URL patterns (only for specific indicators)
	if cfg.Backend == "" && isGradioService(cfg.URL) {
		logger.Info("Detected Gradio TTS service, using GradioClient",
			zap.String("url", cfg.URL))
		return NewGradioClient(cfg, logger)
	}

	// Default to VLLM client for IndexTTS v2 FastAPI services
	logger.Info("Using VLLMClient for IndexTTS API",
		zap.String("url", cfg.URL),
		zap.String("backend", cfg.Backend))
	return NewVLLMClient(cfg, logger)
}

// isGradioService detects if the service URL indicates a Gradio interface.
// NOTE: Only use for auto-detection when backend is not explicitly set.
func isGradioService(url string) bool {
	// Check for explicit Gradio URL patterns only
	// Removed .seetacloud.com as it can host both Gradio and FastAPI services
	gradioIndicators := []string{
		".gradio.live",    // Gradio sharing URLs
		".gradio.app",     // Gradio official app domain
		"/gradio/",        // URL path contains gradio
		":7860",           // Default Gradio port
	}

	for _, indicator := range gradioIndicators {
		if strings.Contains(url, indicator) {
			return true
		}
	}

	return false
}

// LegacyClient handles TTS API calls to the original tts_service.
// Kept for backward compatibility.
type LegacyClient struct {
	baseURL string
	client  *http.Client
	logger  *zap.Logger
}

// NewLegacyClient creates a new legacy TTS client.
func NewLegacyClient(cfg config.TTSConfig, logger *zap.Logger) *LegacyClient {
	return &LegacyClient{
		baseURL: cfg.URL,
		client: &http.Client{
			Timeout: 600 * time.Second,
		},
		logger: logger,
	}
}

// Synthesize performs TTS synthesis using the legacy API.
func (c *LegacyClient) Synthesize(ctx context.Context, req SynthesisRequest) (io.ReadCloser, error) {
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

	// If no URL, assume response body contains audio
	return resp.Body, nil
}
