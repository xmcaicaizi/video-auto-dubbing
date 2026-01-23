// VLLM客户端修改 - 需要替换到 worker/internal/tts/vllm_client.go

package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// 更新indexTTSV2Request结构体，支持情感音频路径
type indexTTSV2Request struct {
	Text                        string    `json:"text"`
	SpkAudioPath                string    `json:"spk_audio_path"`
	EmoControlMethod            int       `json:"emo_control_method,omitempty"`
	EmoRefPath                  string    `json:"emo_ref_path,omitempty"`
	EmoWeight                   float64   `json:"emo_weight,omitempty"`
	EmoVec                      []float64 `json:"emo_vec,omitempty"`
	EmoText                     string    `json:"emo_text,omitempty"`
	EmoRandom                   bool      `json:"emo_random,omitempty"`
	MaxTextTokensPerSentence    int       `json:"max_text_tokens_per_sentence,omitempty"`
	// 新增：支持独立的情感音频路径
	EmoAudioPath                *string   `json:"emo_audio_path,omitempty"`
	EmoAlpha                    float64   `json:"emo_alpha,omitempty"`
}

// 音频上传响应结构
type audioUploadResponse struct {
	ServerPath string `json:"server_path"`
	Filename   string `json:"filename"`
	Size       int64  `json:"size"`
	Status     string `json:"status"`
}

// 音频上传功能
func (c *VLLMClient) uploadPromptAudio(ctx context.Context, audioURL string) (string, error) {
	c.logger.Info("Starting audio upload", zap.String("audio_url", audioURL))

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
	if parsedName := filepath.Base(audioURL); parsedName != "" && parsedName != "/" {
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
	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 设置较长的超时时间，因为音频文件可能较大
	client := &http.Client{Timeout: 30 * time.Second}
	httpResp, err := client.Do(req)
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

// 改造原有的tryIndexTTSV2Endpoint方法
func (c *VLLMClient) tryIndexTTSV2Endpoint(ctx context.Context, req SynthesisRequest) (io.ReadCloser, error) {
	var spkAudioPath string

	// 🔥 核心修复: 上传并使用原始音频进行音色克隆
	if req.PromptAudioURL != "" {
		c.logger.Info("Attempting to use original audio for voice cloning",
			zap.String("prompt_url", req.PromptAudioURL))

		uploaded, err := c.uploadPromptAudio(ctx, req.PromptAudioURL)
		if err != nil {
			c.logger.Warn("Failed to upload prompt audio, using fallback speaker",
				zap.String("url", req.PromptAudioURL),
				zap.Error(err))
			spkAudioPath = c.getFallbackSpeaker(req.SpeakerID)
		} else {
			spkAudioPath = uploaded
			c.logger.Info("Successfully uploaded original audio for voice cloning",
				zap.String("server_path", spkAudioPath))
		}
	} else {
		spkAudioPath = c.getFallbackSpeaker(req.SpeakerID)
		c.logger.Info("No prompt audio provided, using fallback speaker",
			zap.String("speaker_id", req.SpeakerID),
			zap.String("fallback_path", spkAudioPath))
	}

	// 🎵 构建使用音色和情感克隆的请求
	v2Req := c.buildVoiceCloningRequest(req.Text, spkAudioPath, req)

	return c.executeVoiceCloningRequest(ctx, v2Req)
}

// 构建音色克隆请求
func (c *VLLMClient) buildVoiceCloningRequest(text, spkAudioPath string, req SynthesisRequest) indexTTSV2Request {
	baseReq := indexTTSV2Request{
		Text:                     text,
		SpkAudioPath:             spkAudioPath,
		MaxTextTokensPerSentence: 120,
	}

	// 如果上传了原始音频，使用同一个文件作为音色和情感参考
	if req.PromptAudioURL != "" && spkAudioPath != "" && !c.isFallbackSpeaker(spkAudioPath) {
		c.logger.Info("Using same audio for both voice and emotion cloning")
		// 使用音色+情感克隆的新接口
		baseReq.EmoAudioPath = &spkAudioPath  // 情感参考（同一文件）
		baseReq.EmoAlpha = 0.8                // 情感强度
	} else {
		// 降级到仅使用音色参考，情感跟随音色
		c.logger.Info("Using voice reference only, emotion follows voice")
		baseReq.EmoControlMethod = 0 // 情感与音色参考音频相同
	}

	return baseReq
}

// 执行音色克隆请求
func (c *VLLMClient) executeVoiceCloningRequest(ctx context.Context, req indexTTSV2Request) (io.ReadCloser, error) {
	// 优先尝试新的音色克隆接口
	if req.EmoAudioPath != nil {
		return c.tryVoiceCloningEndpoint(ctx, req)
	}

	// 降级到原有接口
	return c.tryOriginalV2Endpoint(ctx, req)
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

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("voice cloning request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("voice cloning endpoint returned status %d", resp.StatusCode)
	}

	c.logger.Info("Voice cloning endpoint success")
	return c.handleVLLMResponse(resp)
}

// 尝试原有的V2接口（降级）
func (c *VLLMClient) tryOriginalV2Endpoint(ctx context.Context, req indexTTSV2Request) (io.ReadCloser, error) {
	bodyBytes, err := json.Marshal(req)
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
		zap.String("spk_audio_path", req.SpkAudioPath))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("IndexTTS v2 request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("IndexTTS v2 returned status %d", resp.StatusCode)
	}

	c.logger.Info("IndexTTS v2 /tts_url fallback success")
	return c.handleVLLMResponse(resp)
}

// 智能预设音色选择（降级机制）
func (c *VLLMClient) getFallbackSpeaker(speakerID string) string {
	speakerMapping := map[string]string{
		"default":      "/root/index-tts-vllm/examples/voice_01.wav",
		"male_young":   "/root/index-tts-vllm/examples/voice_01.wav",
		"female_young": "/root/index-tts-vllm/examples/voice_02.wav",
		"male_mature":  "/root/index-tts-vllm/examples/voice_04.wav",
		"female_mature": "/root/index-tts-vllm/examples/voice_05.wav",
		"speaker_1":    "/root/index-tts-vllm/examples/voice_01.wav",
		"speaker_2":    "/root/index-tts-vllm/examples/voice_02.wav",
		"speaker_3":    "/root/index-tts-vllm/examples/voice_03.wav",
		"speaker_4":    "/root/index-tts-vllm/examples/voice_04.wav",
		"speaker_5":    "/root/index-tts-vllm/examples/voice_05.wav",
	}

	if path, exists := speakerMapping[speakerID]; exists {
		return path
	}
	return speakerMapping["default"]
}

// 检查是否为降级音色
func (c *VLLMClient) isFallbackSpeaker(audioPath string) bool {
	fallbackPaths := []string{
		"/root/index-tts-vllm/examples/voice_01.wav",
		"/root/index-tts-vllm/examples/voice_02.wav",
		"/root/index-tts-vllm/examples/voice_03.wav",
		"/root/index-tts-vllm/examples/voice_04.wav",
		"/root/index-tts-vllm/examples/voice_05.wav",
	}

	for _, fallback := range fallbackPaths {
		if audioPath == fallback {
			return true
		}
	}
	return false
}