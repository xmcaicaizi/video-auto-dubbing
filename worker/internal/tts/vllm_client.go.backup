package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"vedio/shared/config"

	"go.uber.org/zap"
)

// VLLMClient handles TTS API calls to index-tts-vllm service.
// It supports both the native API and OpenAI-compatible API.
type VLLMClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
	logger  *zap.Logger
}

// NewVLLMClient creates a new index-tts-vllm client.
func NewVLLMClient(cfg config.TTSConfig, logger *zap.Logger) *VLLMClient {
	return &VLLMClient{
		baseURL: cfg.URL,
		apiKey:  cfg.APIKey,
		client: &http.Client{
			Timeout: 600 * time.Second,
		},
		logger: logger,
	}
}

// vllmSynthesizeRequest represents the native API request format for index-tts-vllm.
// Based on common TTS API patterns and the project's design.
type vllmSynthesizeRequest struct {
	Text           string   `json:"text"`
	PromptAudio    string   `json:"prompt_audio,omitempty"`     // Base64 encoded or URL
	PromptAudioURL string   `json:"prompt_audio_url,omitempty"` // Alternative: URL to prompt audio
	Speed          float64  `json:"speed,omitempty"`            // Speech speed (default 1.0)
	OutputFormat   string   `json:"output_format,omitempty"`    // wav, mp3
	SampleRate     int      `json:"sample_rate,omitempty"`      // Output sample rate
}

// indexTTSV2Request represents IndexTTS v2 /tts_url API request format.
// Reference: api_example_v2.py
type indexTTSV2Request struct {
	Text                        string    `json:"text"`
	SpkAudioPath                string    `json:"spk_audio_path"`                           // Required: speaker reference audio path
	EmoControlMethod            int       `json:"emo_control_method,omitempty"`             // 0=same as spk, 1=ref audio, 2=vector, 3=text
	EmoRefPath                  string    `json:"emo_ref_path,omitempty"`                   // Emotion reference audio path
	EmoWeight                   float64   `json:"emo_weight,omitempty"`                     // Emotion weight (default 1.0)
	EmoVec                      []float64 `json:"emo_vec,omitempty"`                        // Emotion vector [8 floats]
	EmoText                     string    `json:"emo_text,omitempty"`                       // Emotion description text
	EmoRandom                   bool      `json:"emo_random,omitempty"`                     // Random emotion
	MaxTextTokensPerSentence    int       `json:"max_text_tokens_per_sentence,omitempty"`   // Default 120
}

// vllmSynthesizeResponse represents the native API response format.
type vllmSynthesizeResponse struct {
	Audio      string `json:"audio,omitempty"`       // Base64 encoded audio
	AudioURL   string `json:"audio_url,omitempty"`   // URL to audio file
	DurationMs int    `json:"duration_ms,omitempty"` // Audio duration
	Success    bool   `json:"success,omitempty"`
	Message    string `json:"message,omitempty"`
}

// openAISpeechRequest represents the OpenAI-compatible API request.
type openAISpeechRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float64 `json:"speed,omitempty"`
}

// Synthesize performs TTS synthesis using index-tts-vllm.
// It tries multiple API formats in order of preference:
// 1. Native /synthesize or /tts endpoint
// 2. OpenAI-compatible /audio/speech endpoint
// 3. /v1/audio/speech endpoint
func (c *VLLMClient) Synthesize(ctx context.Context, req SynthesisRequest) (io.ReadCloser, error) {
	// Try native API first
	reader, err := c.synthesizeNative(ctx, req)
	if err == nil {
		return reader, nil
	}
	c.logger.Debug("Native API failed, trying OpenAI-compatible API",
		zap.Error(err),
	)

	// Fallback to OpenAI-compatible API
	reader, err = c.synthesizeOpenAI(ctx, req)
	if err == nil {
		return reader, nil
	}

	return nil, fmt.Errorf("all TTS API attempts failed: %w", err)
}

// synthesizeNative tries the native index-tts-vllm API.
func (c *VLLMClient) synthesizeNative(ctx context.Context, req SynthesisRequest) (io.ReadCloser, error) {
	// Try IndexTTS v2 /tts_url endpoint first (most specific)
	reader, err := c.tryIndexTTSV2Endpoint(ctx, req)
	if err == nil {
		return reader, nil
	}
	c.logger.Debug("IndexTTS v2 /tts_url failed, trying generic endpoints",
		zap.Error(err),
	)

	// Try generic endpoints as fallback
	endpoints := []string{"/synthesize", "/tts", "/api/synthesize", "/api/tts"}

	var lastErr error
	for _, endpoint := range endpoints {
		reader, err := c.tryNativeEndpoint(ctx, endpoint, req)
		if err == nil {
			return reader, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("native API failed: %w", lastErr)
}

// tryIndexTTSV2Endpoint attempts the IndexTTS v2 /tts_url endpoint.
func (c *VLLMClient) tryIndexTTSV2Endpoint(ctx context.Context, req SynthesisRequest) (io.ReadCloser, error) {
	// IndexTTS v2 requires spk_audio_path (server-local file path)
	// Since we use PromptAudioURL (MinIO/OSS URL), we need a mapping strategy

	// Strategy: Use predefined speaker voices on the remote server
	// Map speaker_id to server-local reference audio paths
	speakerMapping := map[string]string{
		"default":   "/root/index-tts-vllm/examples/voice_01.wav",
		"speaker_1": "/root/index-tts-vllm/examples/voice_01.wav",
		"speaker_2": "/root/index-tts-vllm/examples/voice_02.wav",
		"speaker_3": "/root/index-tts-vllm/examples/voice_03.wav",
		"speaker_4": "/root/index-tts-vllm/examples/voice_04.wav",
		"speaker_5": "/root/index-tts-vllm/examples/voice_05.wav",
		"male_1":    "/root/index-tts-vllm/examples/voice_01.wav",
		"male_2":    "/root/index-tts-vllm/examples/voice_04.wav",
		"female_1":  "/root/index-tts-vllm/examples/voice_02.wav",
		"female_2":  "/root/index-tts-vllm/examples/voice_05.wav",
	}

	spkAudioPath := speakerMapping[req.SpeakerID]
	if spkAudioPath == "" {
		spkAudioPath = speakerMapping["default"]
	}

	// Build IndexTTS v2 request
	v2Req := indexTTSV2Request{
		Text:                     req.Text,
		SpkAudioPath:             spkAudioPath,
		EmoControlMethod:         0, // 0 = use speaker audio for emotion too
		MaxTextTokensPerSentence: 120,
	}

	bodyBytes, err := json.Marshal(v2Req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal IndexTTS v2 request: %w", err)
	}

	url := fmt.Sprintf("%s/tts_url", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	c.logger.Debug("Trying IndexTTS v2 /tts_url",
		zap.String("url", url),
		zap.String("speaker", req.SpeakerID),
		zap.String("spk_path", spkAudioPath),
		zap.Int("text_len", len(req.Text)),
	)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("endpoint /tts_url not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// IndexTTS v2 /tts_url returns audio/wav directly
	contentType := resp.Header.Get("Content-Type")
	if isAudioContentType(contentType) {
		c.logger.Info("IndexTTS v2 /tts_url success",
			zap.String("content_type", contentType),
		)
		return resp.Body, nil
	}

	// Unexpected response format
	resp.Body.Close()
	return nil, fmt.Errorf("unexpected content type: %s", contentType)
}

// tryNativeEndpoint attempts a single native API endpoint.
func (c *VLLMClient) tryNativeEndpoint(ctx context.Context, endpoint string, req SynthesisRequest) (io.ReadCloser, error) {
	// Calculate speed from prosody control
	speed := 1.0
	if req.ProsodyControl != nil {
		if s, ok := req.ProsodyControl["speed"].(float64); ok {
			speed = s
		}
	}

	nativeReq := vllmSynthesizeRequest{
		Text:           req.Text,
		PromptAudioURL: req.PromptAudioURL,
		Speed:          speed,
		OutputFormat:   req.OutputFormat,
		SampleRate:     req.SampleRate,
	}

	bodyBytes, err := json.Marshal(nativeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("endpoint not found: %s", endpoint)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Check content type to determine response format
	contentType := resp.Header.Get("Content-Type")

	// If response is audio directly
	if isAudioContentType(contentType) {
		return resp.Body, nil
	}

	// If response is JSON, parse it
	var apiResp vllmSynthesizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	resp.Body.Close()

	// If response contains base64 audio
	if apiResp.Audio != "" {
		audioData, err := base64.StdEncoding.DecodeString(apiResp.Audio)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 audio: %w", err)
		}
		return io.NopCloser(bytes.NewReader(audioData)), nil
	}

	// If response contains audio URL
	if apiResp.AudioURL != "" {
		return c.downloadAudio(ctx, apiResp.AudioURL)
	}

	return nil, fmt.Errorf("no audio in response")
}

// synthesizeOpenAI uses the OpenAI-compatible /audio/speech endpoint.
func (c *VLLMClient) synthesizeOpenAI(ctx context.Context, req SynthesisRequest) (io.ReadCloser, error) {
	// Try multiple possible endpoints
	endpoints := []string{"/audio/speech", "/v1/audio/speech"}

	var lastErr error
	for _, endpoint := range endpoints {
		reader, err := c.tryOpenAIEndpoint(ctx, endpoint, req)
		if err == nil {
			return reader, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("OpenAI API failed: %w", lastErr)
}

// tryOpenAIEndpoint attempts a single OpenAI-compatible endpoint.
func (c *VLLMClient) tryOpenAIEndpoint(ctx context.Context, endpoint string, req SynthesisRequest) (io.ReadCloser, error) {
	speed := 1.0
	if req.ProsodyControl != nil {
		if s, ok := req.ProsodyControl["speed"].(float64); ok {
			speed = s
		}
	}

	voice := req.SpeakerID
	if voice == "" {
		voice = "default"
	}

	format := req.OutputFormat
	if format == "" {
		format = "wav"
	}

	openAIReq := openAISpeechRequest{
		Model:          "index-tts-v2",
		Input:          req.Text,
		Voice:          voice,
		ResponseFormat: format,
		Speed:          speed,
	}

	bodyBytes, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)
	// OpenAI API typically expects audio response directly
	httpReq.Header.Set("Accept", "audio/wav, audio/mpeg, audio/*")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("endpoint not found: %s", endpoint)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// OpenAI /audio/speech returns audio stream directly
	return resp.Body, nil
}

// setHeaders sets common request headers.
func (c *VLLMClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
		req.Header.Set("X-Api-Key", c.apiKey)
	}
}

// downloadAudio downloads audio from a URL.
func (c *VLLMClient) downloadAudio(ctx context.Context, audioURL string) (io.ReadCloser, error) {
	// Handle relative URLs
	if len(audioURL) > 0 && audioURL[0] == '/' {
		audioURL = c.baseURL + audioURL
	}

	req, err := http.NewRequestWithContext(ctx, "GET", audioURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download audio: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to download audio: status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// isAudioContentType checks if the content type indicates audio.
func isAudioContentType(contentType string) bool {
	audioTypes := []string{
		"audio/",
		"application/octet-stream",
	}
	for _, t := range audioTypes {
		if len(contentType) >= len(t) && contentType[:len(t)] == t {
			return true
		}
	}
	return false
}
