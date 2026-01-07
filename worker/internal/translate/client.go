package translate

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

// Client handles translation API calls to GLM.
type Client struct {
	apiKey string
	apiURL string
	model  string
	client *http.Client
	logger *zap.Logger
}

// NewClient creates a new translation client.
func NewClient(cfg config.GLMConfig, logger *zap.Logger) *Client {
	return &Client{
		apiKey: cfg.APIKey,
		apiURL: cfg.APIURL,
		model:  cfg.Model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Translate translates text from source language to target language.
func (c *Client) Translate(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("GLM_API_KEY is required")
	}
	if c.apiURL == "" {
		return nil, fmt.Errorf("GLM_API_URL is required")
	}
	if c.model == "" {
		// Safety fallback (should be set by config defaults)
		c.model = "glm-4.5"
	}
	if len(texts) == 0 {
		return []string{}, nil
	}

	// Build prompt that forces deterministic JSON array output in the same order.
	// We keep the prompt minimal to reduce token usage and failure rate.
	inputJSON, _ := json.Marshal(texts)
	systemPrompt := "你是一个翻译引擎。只输出 JSON 数组（string[]），不要输出任何解释或额外字符。"
	userPrompt := fmt.Sprintf(
		"把下面 JSON 数组中的每个元素从 %s 翻译成 %s，保持顺序与数量一致，只输出 JSON 数组：\n%s",
		sourceLang, targetLang, string(inputJSON),
	)

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

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

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
			req, _ = http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to call translation API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("translation API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response: BigModel chat.completions is OpenAI-like.
	// Be tolerant to message.content being either string or object {type,text}.
	var apiResp struct {
		Choices []struct {
			Message struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("translation API returned no choices")
	}

	contentText, err := decodeGLMContentText(apiResp.Choices[0].Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse translation content: %w", err)
	}
	contentText = strings.TrimSpace(contentText)

	// Expect JSON array of strings
	var out []string
	if err := json.Unmarshal([]byte(contentText), &out); err != nil {
		// Sometimes the model wraps JSON in ```...``` fences; try to strip them.
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

func decodeGLMContentText(raw json.RawMessage) (string, error) {
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
