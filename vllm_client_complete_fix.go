// Worker客户端的完整修改版本
// 文件: worker/internal/tts/vllm_client.go

package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
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
			Timeout: 60 * time.Second, // 增加超时时间，因为音频上传可能较慢
		},
		logger: logger,
	}
}

// SynthesisRequest represents a TTS synthesis request.
type SynthesisRequest struct {
	Text           string `json:"text"`
	SpeakerID      string `json:"speaker_id,omitempty"`
	PromptAudioURL string `json:"prompt_audio_url,omitempty"` // 🔥 关键：原始音频URL
	ResponseFormat string `json:"response_format,omitempty"`
	Speed          float32 `json:"speed,omitempty"`
}

// Reference: api_example_v2.py - 更新结构体支持音色克隆
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
	// 🔥 新增：支持独立的情感音频路径和情感强度
	EmoAudioPath                *string   `json:"emo_audio_path,omitempty"`                 // Optional: separate emotion audio path
	EmoAlpha                    float64   `json:"emo_alpha,omitempty"`                      // Emotion strength (0.0-1.0)
}

// 音频上传响应结构
type audioUploadResponse struct {
	ServerPath string `json:"server_path"`
	Filename   string `json:"filename"`
	Size       int64  `json:"size"`
	Status     string `json:"status"`
}

// vllmSynthesizeResponse represents the native API response format.
type vllmSynthesizeResponse struct {
	Audio      string `json:"audio,omitempty"`       // Base64 encoded audio
	AudioURL   string `json:"audio_url,omitempty"`   // URL to audio file
	DurationMs int    `json:"duration_ms,omitempty"` // Audio duration
	Success    bool   `json:"success,omitempty"`
	Message    string `json:"message,omitempty"`
}

// Synthesize generates speech using the index-tts-vllm API
func (c *VLLMClient) Synthesize(ctx context.Context, req SynthesisRequest) (io.ReadCloser, error) {
	c.logger.Info("Starting TTS synthesis with voice cloning",
		zap.String("text_preview", req.Text[:min(50, len(req.Text))]),
		zap.String("speaker_id", req.SpeakerID),
		zap.String("prompt_audio_url", req.PromptAudioURL),
	)

	// 🔥 优先尝试增强的音色克隆接口
	if req.PromptAudioURL != "" {
		c.logger.Info("Attempting voice cloning with original audio",
			zap.String("audio_url", req.PromptAudioURL))

		audioResp, err := c.tryVoiceCloningWithUpload(ctx, req)
		if err != nil {
			c.logger.Warn("Voice cloning failed, falling back to standard TTS",
				zap.Error(err),
				zap.String("fallback_speaker", req.SpeakerID))
			// 降级到标准TTS
			return c.tryIndexTTSV2Endpoint(ctx, req)
		}

		c.logger.Info("Voice cloning synthesis successful")
		return audioResp, nil
	}

	// 没有原始音频，使用标准TTS
	c.logger.Info("Using standard TTS (no voice cloning)")
	return c.tryIndexTTSV2Endpoint(ctx, req)
}

// 🔥 核心新功能：音色克隆与音频上传
func (c *VLLMClient) tryVoiceCloningWithUpload(ctx context.Context, req SynthesisRequest) (io.ReadCloser, error) {
	// 步骤1：上传原始音频
	serverPath, err := c.uploadPromptAudio(ctx, req.PromptAudioURL)
	if err != nil {
		return nil, fmt.Errorf("failed to upload prompt audio: %w", err)
	}

	c.logger.Info("Audio uploaded successfully, starting voice cloning synthesis",
		zap.String("server_path", serverPath))

	// 步骤2：使用上传的音频进行音色克隆
	return c.executeVoiceCloningRequest(ctx, req, serverPath)
}

// 上传音频到TTS服务器
func (c *VLLMClient) uploadPromptAudio(ctx context.Context, audioURL string) (string, error) {
	c.logger.Debug("Starting audio upload", zap.String("url", audioURL))

	// 1. 下载原音频文件
	resp, err := http.Get(audioURL)
	if err != nil {
		return "", fmt.Errorf("failed to download audio from %s: %w", audioURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download audio, status: %d", resp.StatusCode)
	}

	// 2. 准备multipart上传
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 从URL中提取文件名，或使用默认名称
	filename := "prompt.wav"
	if parsedName := filepath.Base(audioURL); parsedName != "" && parsedName != "/" && parsedName != "." {
		filename = parsedName
	}

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	// 复制音频数据
	copySize, err := io.Copy(part, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to copy audio data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	c.logger.Debug("Audio data prepared for upload",
		zap.Int64("size", copySize),
		zap.String("filename", filename))

	// 3. 上传到TTS服务器
	uploadURL := fmt.Sprintf("%s/upload_audio", c.baseURL)
	uploadReq, err := http.NewRequestWithContext(ctx, "POST", uploadURL, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create upload request: %w", err)
	}

	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())

	// 设置较长的超时时间
	uploadClient := &http.Client{Timeout: 60 * time.Second}
	httpResp, err := uploadClient.Do(uploadReq)
	if err != nil {
		return "", fmt.Errorf("failed to upload audio: %w", err)
	}
	defer httpResp.Body.Close()

	// 4. 处理上传响应
	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return "", fmt.Errorf("upload failed with status %d: %s", httpResp.StatusCode, string(body))
	}

	var result audioUploadResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode upload response: %w", err)
	}

	if result.Status != "success" {
		return "", fmt.Errorf("upload unsuccessful: %s", result.Status)
	}

	c.logger.Info("Audio uploaded successfully",
		zap.String("server_path", result.ServerPath),
		zap.String("filename", result.Filename),
		zap.Int64("size", result.Size),
	)

	return result.ServerPath, nil
}

// 执行音色克隆请求
func (c *VLLMClient) executeVoiceCloningRequest(ctx context.Context, req SynthesisRequest, serverPath string) (io.ReadCloser, error) {
	// 构建音色克隆请求
	v2Req := indexTTSV2Request{
		Text:                     req.Text,
		SpkAudioPath:             serverPath,        // 音色参考
		MaxTextTokensPerSentence: 120,
		EmoAudioPath:             &serverPath,       // 🔥 情感参考（同一文件）
		EmoAlpha:                 0.8,               // 🔥 情感强度
	}

	// 优先尝试增强的音色克隆接口
	return c.tryVoiceCloningEndpoint(ctx, v2Req)
}

// 尝试新的音色克隆接口
func (c *VLLMClient) tryVoiceCloningEndpoint(ctx context.Context, req indexTTSV2Request) (io.ReadCloser, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal voice cloning request: %w", err)
	}

	url := fmt.Sprintf("%s/tts_url_with_cloning", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create voice cloning request: %w", err)
	}

	c.setHeaders(httpReq)

	c.logger.Debug("Calling voice cloning endpoint",
		zap.String("url", url),
		zap.String("spk_audio_path", req.SpkAudioPath),
		zap.String("emo_audio_path", *req.EmoAudioPath),
		zap.Float64("emo_alpha", req.EmoAlpha))

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("voice cloning request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("voice cloning endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("Voice cloning endpoint success")
	return resp.Body, nil
}

// 降级到原有的IndexTTS v2接口
func (c *VLLMClient) tryIndexTTSV2Endpoint(ctx context.Context, req SynthesisRequest) (io.ReadCloser, error) {
	// 使用预设音色
	spkAudioPath := c.getFallbackSpeaker(req.SpeakerID)

	v2Req := indexTTSV2Request{
		Text:                     req.Text,
		SpkAudioPath:             spkAudioPath,
		EmoControlMethod:         0, // 情感与音色参考音频相同
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

	c.logger.Debug("Trying IndexTTS v2 /tts_url (fallback)",
		zap.String("url", url),
		zap.String("spk_audio_path", spkAudioPath))

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("IndexTTS v2 request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("IndexTTS v2 returned status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("IndexTTS v2 /tts_url fallback success")
	return resp.Body, nil
}

// 智能预设音色选择
func (c *VLLMClient) getFallbackSpeaker(speakerID string) string {
	// 🔧 根据你的实际路径调整
	speakerMapping := map[string]string{
		"default":      "/root/index-tts-vllm/examples/voice_01.wav",
		"male_young":   "/root/index-tts-vllm/examples/voice_01.wav",
		"female_young": "/root/index-tts-vllm/examples/voice_01.wav", // 你可以调整为voice_02.wav
		"male_mature":  "/root/index-tts-vllm/examples/voice_01.wav",
		"female_mature": "/root/index-tts-vllm/examples/voice_01.wav",
		"speaker_1":    "/root/index-tts-vllm/examples/voice_01.wav",
		"speaker_2":    "/root/index-tts-vllm/examples/voice_01.wav",
		"speaker_3":    "/root/index-tts-vllm/examples/voice_01.wav",
	}

	if path, exists := speakerMapping[speakerID]; exists {
		return path
	}
	return speakerMapping["default"]
}

// 设置HTTP请求头
func (c *VLLMClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}
}

// 工具函数：返回较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}