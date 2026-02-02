package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"vedio/worker/internal/config"

	"go.uber.org/zap"
)

// DashScopeClient handles translation API calls to Aliyun DashScope (Qwen models).
type DashScopeClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
	logger  *zap.Logger
	limiter *rateLimiter
}

// NewDashScopeClient creates a new DashScope translation client.
func NewDashScopeClient(cfg config.DashScopeLLMConfig, logger *zap.Logger) *DashScopeClient {
	return &DashScopeClient{
		apiKey:  cfg.APIKey,
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:  logger,
		limiter: newRateLimiter(cfg.RPS),
	}
}

// Translate translates text from source language to target language using DashScope API.
func (c *DashScopeClient) Translate(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("DASHSCOPE_LLM_API_KEY is required")
	}
	if c.baseURL == "" {
		return nil, fmt.Errorf("DASHSCOPE_LLM_BASE_URL is required")
	}
	if c.model == "" {
		// Safety fallback (should be set by config defaults)
		c.model = "qwen-turbo"
	}
	if len(texts) == 0 {
		return []string{}, nil
	}
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("translation throttled: %w", err)
		}
	}

	// Build prompt that forces deterministic JSON array output in the same order.
	// Include translation rules to keep output consistent and usable for TTS.
	inputJSON, _ := json.Marshal(texts)
	systemPrompt := strings.Join([]string{
		"你是一个专业的翻译引擎。",
		"只输出 JSON 数组（string[]），不要输出任何解释或额外字符。",
		"保持条目数量与顺序一致，不新增或删除。",
		"保留关键实体（人名、地名、品牌、专有名词）。",
		"数字、日期、货币与单位保持数值含义准确。",
		"标点与格式尽量保留，不添加语气词或无关补充。",
		"翻译需要自然流畅，符合目标语言的表达习惯。",
	}, "")
	userPrompt := fmt.Sprintf(
		"把下面 JSON 数组中的每个元素从 %s 翻译成 %s，保持顺序与数量一致，只输出 JSON 数组：\n%s",
		sourceLang, targetLang, string(inputJSON),
	)

	// Construct request URL - DashScope uses /chat/completions endpoint
	apiURL := strings.TrimSuffix(c.baseURL, "/") + "/chat/completions"

	reqBody := map[string]interface{}{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.2,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request with retry/backoff to avoid rate limit failures
	var resp *http.Response
	maxRetries := 6
	var lastStatus int
	for i := 0; i < maxRetries; i++ {
		// Create HTTP request
		req, reqErr := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bodyBytes))
		if reqErr != nil {
			return nil, fmt.Errorf("failed to create request: %w", reqErr)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

		resp, err = c.client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil {
			lastStatus = resp.StatusCode
			if shouldRetryStatus(resp.StatusCode) && i < maxRetries-1 {
				delay := retryDelay(resp, i)
				resp.Body.Close()
				c.logger.Warn("Retrying DashScope translation request",
					zap.Int("attempt", i+1),
					zap.Int("status", resp.StatusCode),
					zap.Duration("delay", delay),
				)
				time.Sleep(delay)
				continue
			}
		}
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to call DashScope translation API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if lastStatus != 0 {
			return nil, fmt.Errorf("DashScope API returned status %d: %s", lastStatus, string(body))
		}
		return nil, fmt.Errorf("DashScope API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response: DashScope compatible-mode follows OpenAI format
	var apiResp struct {
		Choices []struct {
			Message struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode DashScope response: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("DashScope API returned no choices")
	}

	contentText, err := decodeContentText(apiResp.Choices[0].Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DashScope content: %w", err)
	}
	contentText = strings.TrimSpace(contentText)

	// Expect JSON array of strings
	var out []string
	if err := json.Unmarshal([]byte(contentText), &out); err != nil {
		// Sometimes the model wraps JSON in ```...``` fences; try to strip them
		clean := stripCodeFences(contentText)
		if clean != contentText {
			if err2 := json.Unmarshal([]byte(clean), &out); err2 != nil {
				return nil, fmt.Errorf("expected JSON string array, got: %s", truncate(contentText, 300))
			}
		} else {
			return nil, fmt.Errorf("expected JSON string array, got: %s", truncate(contentText, 300))
		}
	}
	if len(out) != len(texts) {
		return nil, fmt.Errorf("translation count mismatch: expected %d, got %d", len(texts), len(out))
	}
	return out, nil
}

func decodeContentText(raw json.RawMessage) (string, error) {
	// Try string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	// Try object form {type,text}
	var obj struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Text != "" {
		return obj.Text, nil
	}
	return "", fmt.Errorf("unsupported content format")
}

func shouldRetryStatus(status int) bool {
	return status == http.StatusTooManyRequests ||
		status == http.StatusServiceUnavailable ||
		status == http.StatusBadGateway ||
		status == http.StatusGatewayTimeout
}

func retryDelay(resp *http.Response, attempt int) time.Duration {
	if resp != nil {
		if v := resp.Header.Get("Retry-After"); v != "" {
			if seconds, err := strconv.Atoi(v); err == nil && seconds > 0 {
				return time.Duration(seconds) * time.Second
			}
		}
	}
	// Exponential backoff with jitter for rate-limit responses
	base := time.Duration(5*(attempt+1)) * time.Second
	jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
	return base + jitter
}

func stripCodeFences(s string) string {
	trim := strings.TrimSpace(s)
	if strings.HasPrefix(trim, "```") {
		trim = strings.TrimPrefix(trim, "```")
		trim = strings.TrimSpace(trim)
		// Optional language tag line
		if i := strings.IndexByte(trim, '\n'); i != -1 {
			firstLine := strings.TrimSpace(trim[:i])
			rest := trim[i+1:]
			if firstLine == "" || len(firstLine) <= 10 {
				trim = rest
			}
		}
		if j := strings.LastIndex(trim, "```"); j != -1 {
			trim = trim[:j]
		}
	}
	return strings.TrimSpace(trim)
}

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
