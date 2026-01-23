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

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	volcengineSubmitURL = "https://openspeech.bytedance.com/api/v3/auc/bigmodel/submit"
	volcengineQueryURL  = "https://openspeech.bytedance.com/api/v3/auc/bigmodel/query"

	// Status codes
	statusSuccess    = 20000000
	statusProcessing = 20000001
	statusQueued     = 20000002
	statusSilence    = 20000003
)

// VolcengineClient handles ASR API calls to Volcengine (火山引擎) service.
type VolcengineClient struct {
	cfg    config.VolcengineASRConfig
	client *http.Client
	logger *zap.Logger
}

// NewVolcengineClient creates a new Volcengine ASR client.
func NewVolcengineClient(cfg config.VolcengineASRConfig, logger *zap.Logger) *VolcengineClient {
	return &VolcengineClient{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second, // Per-request timeout
		},
		logger: logger,
	}
}

// submitRequest represents the request body for task submission.
type submitRequest struct {
	User    submitUser    `json:"user"`
	Audio   submitAudio   `json:"audio"`
	Request submitOptions `json:"request"`
}

type submitUser struct {
	UID string `json:"uid"`
}

type submitAudio struct {
	Format  string `json:"format"`
	URL     string `json:"url"`
	Rate    int    `json:"rate,omitempty"`
	Channel int    `json:"channel,omitempty"`
}

type submitOptions struct {
	ModelName             string `json:"model_name"`
	ModelVersion          string `json:"model_version,omitempty"`
	EnableITN             bool   `json:"enable_itn"`
	EnablePunc            bool   `json:"enable_punc"`
	EnableSpeakerInfo     bool   `json:"enable_speaker_info"`
	EnableEmotionDetect   bool   `json:"enable_emotion_detection"`
	EnableGenderDetect    bool   `json:"enable_gender_detection"`
	ShowUtterances        bool   `json:"show_utterances"`
}

// queryResponse represents the response from query API.
type queryResponse struct {
	AudioInfo struct {
		Duration int `json:"duration"`
	} `json:"audio_info"`
	Result struct {
		Text       string      `json:"text"`
		Utterances []utterance `json:"utterances"`
	} `json:"result"`
}

type utterance struct {
	Text      string     `json:"text"`
	StartTime int        `json:"start_time"`
	EndTime   int        `json:"end_time"`
	Additions *additions `json:"additions,omitempty"`
	Words     []word     `json:"words,omitempty"`
}

type additions struct {
	SpeakerID  string  `json:"speaker_id,omitempty"`
	Emotion    string  `json:"emotion,omitempty"`
	Gender     string  `json:"gender,omitempty"`
	SpeechRate float64 `json:"speech_rate,omitempty"`
}

type word struct {
	Text      string `json:"text"`
	StartTime int    `json:"start_time"`
	EndTime   int    `json:"end_time"`
}

// Recognize performs ASR using the Volcengine service.
// It submits the task and polls for the result.
func (c *VolcengineClient) Recognize(ctx context.Context, audioURL string, language string) (*models.ASRResult, error) {
	// Generate unique request ID
	requestID := uuid.New().String()

	c.logger.Info("Submitting ASR task to Volcengine",
		zap.String("request_id", requestID),
		zap.String("audio_url", audioURL),
		zap.String("language", language),
	)

	// Submit task
	if err := c.submitTask(ctx, requestID, audioURL, language); err != nil {
		return nil, fmt.Errorf("failed to submit ASR task: %w", err)
	}

	// Poll for result
	result, err := c.pollResult(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ASR result: %w", err)
	}

	return result, nil
}

// submitTask submits an ASR task to Volcengine.
func (c *VolcengineClient) submitTask(ctx context.Context, requestID, audioURL, language string) error {
	reqBody := submitRequest{
		User: submitUser{
			UID: requestID,
		},
		Audio: submitAudio{
			Format:  "wav",
			URL:     audioURL,
			Rate:    16000,
			Channel: 1,
		},
		Request: submitOptions{
			ModelName:           "bigmodel",
			ModelVersion:        "400",
			EnableITN:           c.cfg.EnableITN,
			EnablePunc:          c.cfg.EnablePunc,
			EnableSpeakerInfo:   c.cfg.EnableSpeakerInfo,
			EnableEmotionDetect: c.cfg.EnableEmotionDetect,
			EnableGenderDetect:  c.cfg.EnableGenderDetect,
			ShowUtterances:      true,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", volcengineSubmitURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req, requestID)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	statusCode := resp.Header.Get("X-Api-Status-Code")
	message := resp.Header.Get("X-Api-Message")

	if statusCode != "20000000" {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("submit failed with status %s: %s, body: %s", statusCode, message, string(body))
	}

	c.logger.Info("ASR task submitted successfully",
		zap.String("request_id", requestID),
	)

	return nil
}

// pollResult polls for the ASR result until completion or timeout.
func (c *VolcengineClient) pollResult(ctx context.Context, requestID string) (*models.ASRResult, error) {
	pollInterval := time.Duration(c.cfg.PollIntervalSeconds) * time.Second
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}

	pollTimeout := time.Duration(c.cfg.PollTimeoutSeconds) * time.Second
	if pollTimeout <= 0 {
		pollTimeout = 15 * time.Minute
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, pollTimeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("ASR polling timeout after %v", pollTimeout)
			}
			return nil, timeoutCtx.Err()
		case <-ticker.C:
			result, statusCode, err := c.queryTask(ctx, requestID)
			if err != nil {
				return nil, err
			}

			switch statusCode {
			case statusSuccess:
				c.logger.Info("ASR task completed",
					zap.String("request_id", requestID),
					zap.Int("segment_count", len(result.Segments)),
				)
				return result, nil
			case statusProcessing, statusQueued:
				c.logger.Debug("ASR task still processing",
					zap.String("request_id", requestID),
					zap.Int("status_code", statusCode),
				)
				continue
			case statusSilence:
				c.logger.Warn("ASR detected silence audio",
					zap.String("request_id", requestID),
				)
				return &models.ASRResult{Segments: []models.ASRSegment{}}, nil
			default:
				return nil, fmt.Errorf("ASR failed with status code %d", statusCode)
			}
		}
	}
}

// queryTask queries the status and result of an ASR task.
func (c *VolcengineClient) queryTask(ctx context.Context, requestID string) (*models.ASRResult, int, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", volcengineQueryURL, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create query request: %w", err)
	}

	c.setHeaders(req, requestID)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to send query request: %w", err)
	}
	defer resp.Body.Close()

	statusCodeStr := resp.Header.Get("X-Api-Status-Code")
	var statusCode int
	if _, err := fmt.Sscanf(statusCodeStr, "%d", &statusCode); err != nil {
		return nil, 0, fmt.Errorf("failed to parse status code: %s", statusCodeStr)
	}

	// If still processing, return early
	if statusCode == statusProcessing || statusCode == statusQueued || statusCode == statusSilence {
		return nil, statusCode, nil
	}

	// If not success, return error
	if statusCode != statusSuccess {
		message := resp.Header.Get("X-Api-Message")
		return nil, statusCode, fmt.Errorf("query failed with status %d: %s", statusCode, message)
	}

	// Parse response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, statusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	var queryResp queryResponse
	if err := json.Unmarshal(body, &queryResp); err != nil {
		return nil, statusCode, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to ASRResult
	result := c.convertToASRResult(&queryResp)

	return result, statusCode, nil
}

// convertToASRResult converts Volcengine response to internal ASRResult format.
func (c *VolcengineClient) convertToASRResult(resp *queryResponse) *models.ASRResult {
	segments := make([]models.ASRSegment, 0, len(resp.Result.Utterances))

	// 🔍 调试日志：检查火山引擎返回的说话人信息
	speakerCount := 0
	for _, utt := range resp.Result.Utterances {
		if utt.Additions != nil && utt.Additions.SpeakerID != "" {
			speakerCount++
		}
	}
	c.logger.Info("Volcengine ASR speaker info analysis",
		zap.Int("total_utterances", len(resp.Result.Utterances)),
		zap.Int("with_speaker_id", speakerCount))

	for idx, utt := range resp.Result.Utterances {
		seg := models.ASRSegment{
			Idx:     idx,
			StartMs: utt.StartTime,
			EndMs:   utt.EndTime,
			Text:    utt.Text,
		}

		if utt.Additions != nil {
			seg.SpeakerID = utt.Additions.SpeakerID
			seg.Emotion = utt.Additions.Emotion
			seg.Gender = utt.Additions.Gender
		}

		// 🔥 确保总是有说话人ID以启用音色克隆功能
		if seg.SpeakerID == "" {
			seg.SpeakerID = "speaker_1" // 默认说话人，将触发音色克隆
		}

		segments = append(segments, seg)
	}

	return &models.ASRResult{
		Language:   "", // Volcengine doesn't return language in response
		DurationMs: resp.AudioInfo.Duration,
		Segments:   segments,
	}
}

// setHeaders sets the required headers for Volcengine API requests.
func (c *VolcengineClient) setHeaders(req *http.Request, requestID string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-App-Key", c.cfg.AppKey)
	req.Header.Set("X-Api-Access-Key", c.cfg.AccessKey)
	req.Header.Set("X-Api-Resource-Id", c.cfg.ResourceID)
	req.Header.Set("X-Api-Request-Id", requestID)
	req.Header.Set("X-Api-Sequence", "-1")
}
