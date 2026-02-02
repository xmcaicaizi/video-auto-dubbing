package asr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"vedio/shared/config"
	"vedio/worker/internal/models"

	"go.uber.org/zap"
)

const (
	// 阿里云百炼平台 DashScope ASR 异步API 端点（中国大陆）
	aliyunASRSubmitURL = "https://dashscope.aliyuncs.com/api/v1/services/audio/asr/transcription"
	aliyunASRQueryURL  = "https://dashscope.aliyuncs.com/api/v1/tasks"
)

// AliyunASRConfig holds Aliyun DashScope ASR configuration.
type AliyunASRConfig struct {
	APIKey              string // DashScope API Key
	Model               string // 模型名称，默认 qwen3-asr-flash-filetrans
	EnableITN           bool   // 启用文本规范化（仅支持中英文）
	EnableWords         bool   // 启用词级时间戳
	Language            string // 语言代码（可选）：zh, en, yue, ja 等
	RequestTimeout      int    // 单次请求超时时间（秒）
	PollIntervalSeconds int    // 轮询间隔（秒）
	PollTimeoutSeconds  int    // 轮询总超时（秒）
}

// AliyunClient handles ASR API calls to Aliyun DashScope service.
type AliyunClient struct {
	cfg    AliyunASRConfig
	client *http.Client
	logger *zap.Logger
}

// NewAliyunClient creates a new Aliyun DashScope ASR client.
func NewAliyunClient(cfg AliyunASRConfig, logger *zap.Logger) *AliyunClient {
	timeout := time.Duration(cfg.RequestTimeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// 设置默认值
	if cfg.PollIntervalSeconds <= 0 {
		cfg.PollIntervalSeconds = 2
	}
	if cfg.PollTimeoutSeconds <= 0 {
		cfg.PollTimeoutSeconds = 900 // 15分钟
	}
	if cfg.Model == "" {
		cfg.Model = "qwen3-asr-flash-filetrans"
	}

	return &AliyunClient{
		cfg: cfg,
		client: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// aliyunASRRequest represents the request body for DashScope ASR API.
type aliyunASRRequest struct {
	Model      string                   `json:"model"`
	Input      aliyunASRInput           `json:"input"`
	Parameters *aliyunASRParameters     `json:"parameters,omitempty"`
}

type aliyunASRInput struct {
	Messages []aliyunASRMessage `json:"messages"`
}

type aliyunASRMessage struct {
	Role    string                `json:"role"`
	Content []aliyunASRContent    `json:"content"`
}

type aliyunASRContent struct {
	Audio string `json:"audio"` // 音频URL或Base64编码
}

type aliyunASRParameters struct {
	ASROptions *aliyunASROptions `json:"asr_options,omitempty"`
}

type aliyunASROptions struct {
	Language  string `json:"language,omitempty"`  // 语言代码
	EnableITN bool   `json:"enable_itn,omitempty"` // 文本规范化
}

// aliyunASRResponse represents the response from DashScope ASR API.
type aliyunASRResponse struct {
	Output struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Role    string `json:"role"`
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
				Annotations []struct {
					Type     string `json:"type"`
					Language string `json:"language,omitempty"`
					Emotion  string `json:"emotion,omitempty"`
				} `json:"annotations,omitempty"`
			} `json:"message"`
		} `json:"choices"`
	} `json:"output"`
	Usage struct {
		InputTokensDetails struct {
			TextTokens int `json:"text_tokens"`
		} `json:"input_tokens_details"`
		OutputTokensDetails struct {
			TextTokens int `json:"text_tokens"`
		} `json:"output_tokens_details"`
		Seconds int `json:"seconds"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

// Recognize performs ASR using the Aliyun DashScope service.
func (c *AliyunClient) Recognize(ctx context.Context, audioURL string, language string) (*models.ASRResult, error) {
	c.logger.Info("Starting Aliyun DashScope ASR recognition",
		zap.String("audio_url", audioURL),
		zap.String("language", language),
		zap.String("model", c.cfg.Model),
	)

	// 构建请求
	reqBody := c.buildRequest(audioURL, language)

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", aliyunASREndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.cfg.APIKey))

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ASR request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var asrResp aliyunASRResponse
	if err := json.Unmarshal(body, &asrResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, body: %s", err, string(body))
	}

	// 检查响应有效性
	if len(asrResp.Output.Choices) == 0 {
		return nil, fmt.Errorf("no choices in ASR response")
	}

	// 转换为内部格式
	result := c.convertToASRResult(&asrResp)

	c.logger.Info("Aliyun DashScope ASR recognition completed",
		zap.String("request_id", asrResp.RequestID),
		zap.Int("duration_seconds", asrResp.Usage.Seconds),
		zap.String("detected_language", result.Language),
		zap.Int("segment_count", len(result.Segments)),
	)

	return result, nil
}

// buildRequest constructs the ASR request payload.
func (c *AliyunClient) buildRequest(audioURL string, language string) *aliyunASRRequest {
	req := &aliyunASRRequest{
		Model: c.cfg.Model,
		Input: aliyunASRInput{
			Messages: []aliyunASRMessage{
				{
					Role: "user",
					Content: []aliyunASRContent{
						{
							Audio: audioURL,
						},
					},
				},
			},
		},
	}

	// 添加 ASR 选项（如果需要）
	if c.cfg.EnableITN || language != "" || c.cfg.Language != "" {
		req.Parameters = &aliyunASRParameters{
			ASROptions: &aliyunASROptions{},
		}

		if c.cfg.EnableITN {
			req.Parameters.ASROptions.EnableITN = true
		}

		// 优先使用传入的 language 参数
		if language != "" {
			req.Parameters.ASROptions.Language = language
		} else if c.cfg.Language != "" {
			req.Parameters.ASROptions.Language = c.cfg.Language
		}
	}

	return req
}

// convertToASRResult converts Aliyun DashScope response to internal ASRResult format.
// 注意：Qwen ASR API 返回的是完整文本，不包含时间戳信息。
// 如需时间戳，需要使用异步文件转写 API（qwen3-asr-flash-filetrans）。
func (c *AliyunClient) convertToASRResult(resp *aliyunASRResponse) *models.ASRResult {
	result := &models.ASRResult{
		Segments:   make([]models.ASRSegment, 0),
		Language:   "",
		DurationMs: resp.Usage.Seconds * 1000,
	}

	if len(resp.Output.Choices) == 0 {
		return result
	}

	choice := resp.Output.Choices[0]

	// 提取语言信息
	for _, annotation := range choice.Message.Annotations {
		if annotation.Type == "audio_info" && annotation.Language != "" {
			result.Language = annotation.Language
			break
		}
	}

	// 提取文本内容
	if len(choice.Message.Content) > 0 {
		fullText := choice.Message.Content[0].Text

		// 由于 Qwen ASR 同步 API 不提供时间戳，我们创建一个单一的 segment
		// 涵盖整个音频的时长
		segment := models.ASRSegment{
			Idx:       0,
			StartMs:   0,
			EndMs:     result.DurationMs,
			Text:      fullText,
			SpeakerID: "speaker_1", // 默认说话人 ID
		}

		// 提取情绪信息（如果有）
		for _, annotation := range choice.Message.Annotations {
			if annotation.Type == "audio_info" && annotation.Emotion != "" {
				segment.Emotion = annotation.Emotion
				break
			}
		}

		result.Segments = append(result.Segments, segment)
	}

	return result
}
