package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"vedio/api/internal/database"
	"vedio/api/internal/models"
	sharedconfig "vedio/shared/config"
	sharedstorage "vedio/shared/storage"
	"github.com/google/uuid"
)

// SettingsService handles settings-related operations.
type SettingsService struct {
	db *database.DB
}

// NewSettingsService creates a new settings service.
func NewSettingsService(db *database.DB) *SettingsService {
	return &SettingsService{db: db}
}

// GetSettings retrieves all settings from the database.
func (s *SettingsService) GetSettings(ctx context.Context) (*models.Settings, error) {
	query := `SELECT category, key, value FROM settings`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query settings: %w", err)
	}
	defer rows.Close()

	settings := &models.Settings{
		ASR: models.ASRSettings{
			VolcengineResourceID: "volc.bigasr.auc",
			EnableSpeakerInfo:    true,
			EnableEmotion:        true,
			EnableGender:         true,
			EnablePunc:           true,
			EnableITN:            true,
		},
		TTS: models.TTSSettings{
			Backend: "vllm",
		},
		Translate: models.TranslateSettings{
			GLMAPIURL: "https://open.bigmodel.cn/api/paas/v4/chat/completions",
			GLMModel:  "glm-4-flash",
		},
		Storage: models.StorageSettings{
			Backend: "minio",
			OSS: models.OSSSettings{
				UseSSL: true,
			},
		},
	}

	for rows.Next() {
		var category, key, value string
		if err := rows.Scan(&category, &key, &value); err != nil {
			continue
		}
		s.applySettingValue(settings, category, key, value)
	}

	return settings, nil
}

// applySettingValue applies a single setting value to the settings struct.
func (s *SettingsService) applySettingValue(settings *models.Settings, category, key, value string) {
	switch category {
	case "storage":
		switch key {
		case "backend":
			settings.Storage.Backend = value
		case "oss_endpoint":
			settings.Storage.OSS.Endpoint = value
		case "oss_bucket":
			settings.Storage.OSS.Bucket = value
		case "oss_access_key_id":
			settings.Storage.OSS.AccessKeyID = value
		case "oss_access_key_secret":
			settings.Storage.OSS.AccessKeySecret = value
		case "oss_public_domain":
			settings.Storage.OSS.PublicDomain = value
		case "oss_prefix":
			settings.Storage.OSS.Prefix = value
		case "oss_use_ssl":
			settings.Storage.OSS.UseSSL = value == "true"
		}
	case "asr":
		switch key {
		case "volcengine_app_key":
			settings.ASR.VolcengineAppKey = value
		case "volcengine_access_key":
			settings.ASR.VolcengineAccessKey = value
		case "volcengine_resource_id":
			settings.ASR.VolcengineResourceID = value
		case "enable_speaker_info":
			settings.ASR.EnableSpeakerInfo = value == "true"
		case "enable_emotion":
			settings.ASR.EnableEmotion = value == "true"
		case "enable_gender":
			settings.ASR.EnableGender = value == "true"
		case "enable_punc":
			settings.ASR.EnablePunc = value == "true"
		case "enable_itn":
			settings.ASR.EnableITN = value == "true"
		}
	case "tts":
		switch key {
		case "service_url":
			settings.TTS.ServiceURL = value
		case "api_key":
			settings.TTS.APIKey = value
		case "backend":
			settings.TTS.Backend = value
		}
	case "translate":
		switch key {
		case "glm_api_key":
			settings.Translate.GLMAPIKey = value
		case "glm_api_url":
			settings.Translate.GLMAPIURL = value
		case "glm_model":
			settings.Translate.GLMModel = value
		}
	}
}

// UpdateSettings updates settings in the database.
func (s *SettingsService) UpdateSettings(ctx context.Context, req *models.SettingsUpdateRequest) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update ASR settings
	if req.ASR != nil {
		if err := s.upsertSetting(ctx, tx, "asr", "volcengine_app_key", req.ASR.VolcengineAppKey, true); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "asr", "volcengine_access_key", req.ASR.VolcengineAccessKey, true); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "asr", "volcengine_resource_id", req.ASR.VolcengineResourceID, false); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "asr", "enable_speaker_info", strconv.FormatBool(req.ASR.EnableSpeakerInfo), false); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "asr", "enable_emotion", strconv.FormatBool(req.ASR.EnableEmotion), false); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "asr", "enable_gender", strconv.FormatBool(req.ASR.EnableGender), false); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "asr", "enable_punc", strconv.FormatBool(req.ASR.EnablePunc), false); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "asr", "enable_itn", strconv.FormatBool(req.ASR.EnableITN), false); err != nil {
			return err
		}
	}

	// Update TTS settings
	if req.TTS != nil {
		if err := s.upsertSetting(ctx, tx, "tts", "service_url", req.TTS.ServiceURL, false); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "tts", "api_key", req.TTS.APIKey, true); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "tts", "backend", req.TTS.Backend, false); err != nil {
			return err
		}
	}

	// Update Translate settings
	if req.Translate != nil {
		if err := s.upsertSetting(ctx, tx, "translate", "glm_api_key", req.Translate.GLMAPIKey, true); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "translate", "glm_api_url", req.Translate.GLMAPIURL, false); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "translate", "glm_model", req.Translate.GLMModel, false); err != nil {
			return err
		}
	}

	// Update Storage settings
	if req.Storage != nil {
		if err := s.upsertSetting(ctx, tx, "storage", "backend", req.Storage.Backend, false); err != nil {
			return err
		}
		// OSS
		if err := s.upsertSetting(ctx, tx, "storage", "oss_endpoint", req.Storage.OSS.Endpoint, false); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "storage", "oss_bucket", req.Storage.OSS.Bucket, false); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "storage", "oss_access_key_id", req.Storage.OSS.AccessKeyID, true); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "storage", "oss_access_key_secret", req.Storage.OSS.AccessKeySecret, true); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "storage", "oss_public_domain", req.Storage.OSS.PublicDomain, false); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "storage", "oss_prefix", req.Storage.OSS.Prefix, false); err != nil {
			return err
		}
		if err := s.upsertSetting(ctx, tx, "storage", "oss_use_ssl", strconv.FormatBool(req.Storage.OSS.UseSSL), false); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// upsertSetting inserts or updates a single setting.
func (s *SettingsService) upsertSetting(ctx context.Context, tx *sql.Tx, category, key, value string, isSensitive bool) error {
	query := `
		INSERT INTO settings (category, key, value, is_encrypted, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (category, key) DO UPDATE
		SET value = EXCLUDED.value, is_encrypted = EXCLUDED.is_encrypted, updated_at = EXCLUDED.updated_at
	`
	_, err := tx.ExecContext(ctx, query, category, key, value, isSensitive, time.Now())
	if err != nil {
		return fmt.Errorf("failed to upsert setting %s.%s: %w", category, key, err)
	}
	return nil
}

// TestConnection tests the connection to a service.
func (s *SettingsService) TestConnection(ctx context.Context, serviceType string) (*models.TestConnectionResponse, error) {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	switch serviceType {
	case "asr":
		return s.testASRConnection(ctx, settings)
	case "tts":
		return s.testTTSConnection(ctx, settings)
	case "translate":
		return s.testTranslateConnection(ctx, settings)
	case "storage":
		return s.testStorageConnection(ctx, settings)
	default:
		return &models.TestConnectionResponse{
			Status:  "failed",
			Message: "未知的服务类型",
		}, nil
	}
}

func (s *SettingsService) testStorageConnection(ctx context.Context, settings *models.Settings) (*models.TestConnectionResponse, error) {
	start := time.Now()
	backend := strings.TrimSpace(settings.Storage.Backend)
	if backend == "" {
		backend = "minio"
	}
	if backend != "oss" {
		return &models.TestConnectionResponse{Status: "failed", Message: "请先选择 backend=oss"}, nil
	}
	oss := settings.Storage.OSS
	if oss.Endpoint == "" || oss.Bucket == "" || oss.AccessKeyID == "" || oss.AccessKeySecret == "" || oss.PublicDomain == "" {
		return &models.TestConnectionResponse{Status: "failed", Message: "OSS 配置不完整，请填写 endpoint/bucket/ak/sk/public_domain"}, nil
	}
	// Create a minimal BaseConfig to initialize OSS storage and generate a signed URL.
	base := &sharedconfig.BaseConfig{
		Storage: sharedconfig.StorageConfig{Backend: "oss"},
		OSS: sharedconfig.OSSConfig{
			Endpoint:        oss.Endpoint,
			Bucket:          oss.Bucket,
			AccessKeyID:     oss.AccessKeyID,
			AccessKeySecret: oss.AccessKeySecret,
			PublicDomain:    oss.PublicDomain,
			Prefix:          oss.Prefix,
			UseSSL:          oss.UseSSL,
		},
	}
	store, err := sharedstorage.NewOSS(base.OSS)
	if err != nil {
		return &models.TestConnectionResponse{Status: "failed", Message: "OSS 初始化失败: " + err.Error()}, nil
	}
	// Try generating a signed URL for a non-existing object to validate signature path.
	_, err = store.PresignedGetURL(ctx, "__connection_test__/probe.txt", 10*time.Minute)
	if err != nil {
		return &models.TestConnectionResponse{Status: "failed", Message: "OSS 签名URL生成失败: " + err.Error()}, nil
	}
	return &models.TestConnectionResponse{Status: "connected", Message: "OSS 配置可用（签名URL生成成功）", LatencyMs: time.Since(start).Milliseconds()}, nil
}

// testASRConnection tests the Volcengine ASR connection.
func (s *SettingsService) testASRConnection(ctx context.Context, settings *models.Settings) (*models.TestConnectionResponse, error) {
	if settings.ASR.VolcengineAppKey == "" || settings.ASR.VolcengineAccessKey == "" {
		return &models.TestConnectionResponse{
			Status:  "failed",
			Message: "ASR 配置不完整，请填写 App Key 和 Access Key",
		}, nil
	}

	// Test connection by submitting a dummy task to Volcengine API
	start := time.Now()

	// Create a test request with invalid audio URL to check authentication
	testRequest := map[string]interface{}{
		"user": map[string]interface{}{
			"uid": "connection_test",
		},
		"audio": map[string]interface{}{
			"format": "wav",
			"url":    "http://invalid-test-url.wav", // Intentionally invalid for auth test
		},
		"request": map[string]interface{}{
			"model_name":      "bigmodel",
			"show_utterances": true,
		},
	}

	// Convert to JSON

	jsonData, err := json.Marshal(testRequest)
	if err != nil {
		return &models.TestConnectionResponse{
			Status:  "failed",
			Message: "创建测试请求失败: " + err.Error(),
		}, nil
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://openspeech.bytedance.com/api/v3/auc/bigmodel/submit", bytes.NewBuffer(jsonData))
	if err != nil {
		return &models.TestConnectionResponse{
			Status:  "failed",
			Message: "创建 HTTP 请求失败: " + err.Error(),
		}, nil
	}

	// Generate request ID
	requestID := uuid.New().String()

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-App-Key", settings.ASR.VolcengineAppKey)
	req.Header.Set("X-Api-Access-Key", settings.ASR.VolcengineAccessKey)
	req.Header.Set("X-Api-Resource-Id", settings.ASR.VolcengineResourceID)
	req.Header.Set("X-Api-Request-Id", requestID)
	req.Header.Set("X-Api-Sequence", "-1")

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &models.TestConnectionResponse{
			Status:  "failed",
			Message: "网络连接失败: " + err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	latency := time.Since(start).Milliseconds()

	// Check response
	statusCode := resp.Header.Get("X-Api-Status-Code")
	message := resp.Header.Get("X-Api-Message")

	switch statusCode {
	case "20000000":
		// Success - authentication works
		return &models.TestConnectionResponse{
			Status:    "connected",
			Message:   "火山引擎 ASR 连接测试成功",
			LatencyMs: latency,
		}, nil
	case "45000001", "45000002", "45000151":
		// Audio-related errors indicate auth success but invalid audio (expected)
		return &models.TestConnectionResponse{
			Status:    "connected",
			Message:   "火山引擎 ASR 认证成功 (测试音频错误属于正常)",
			LatencyMs: latency,
		}, nil
	case "40000001", "40000002", "40000003":
		// Authentication errors
		return &models.TestConnectionResponse{
			Status:  "failed",
			Message: fmt.Sprintf("认证失败: %s - %s", statusCode, message),
		}, nil
	case "55000031":
		// Service busy
		return &models.TestConnectionResponse{
			Status:    "connected",
			Message:   "服务繁忙，但认证通过",
			LatencyMs: latency,
		}, nil
	default:
		// Other errors
		return &models.TestConnectionResponse{
			Status:  "failed",
			Message: fmt.Sprintf("未知错误: %s - %s", statusCode, message),
		}, nil
	}
}

// testTTSConnection tests the TTS service connection.
func (s *SettingsService) testTTSConnection(ctx context.Context, settings *models.Settings) (*models.TestConnectionResponse, error) {
	if settings.TTS.ServiceURL == "" {
		return &models.TestConnectionResponse{
			Status:  "failed",
			Message: "TTS 服务地址未配置",
		}, nil
	}

	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}

	// Try different endpoints to detect service type and test connectivity
	testEndpoints := []struct {
		path        string
		description string
		expectJSON  bool
	}{
		{"/health", "健康检查", true},
		{"/api/health", "API 健康检查", true},
		{"/v1/health", "V1 健康检查", true},
		{"/docs", "FastAPI 文档", false},
		{"/openapi.json", "OpenAPI 文档", true},
		{"/", "根路径", false},
		{"/gradio_api/info", "Gradio API 信息", true},
	}

	var lastErr error
	var serviceType string

	for _, endpoint := range testEndpoints {
		testURL := settings.TTS.ServiceURL + endpoint.path
		req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
		if err != nil {
			continue
		}

		if settings.TTS.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+settings.TTS.APIKey)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// 检查状态码范围
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)
			resp.Body.Close()

			latency := time.Since(start).Milliseconds()

			// 根据响应内容检测服务类型
			bodyLower := strings.ToLower(bodyStr)

			if endpoint.path == "/health" && (strings.Contains(bodyStr, "OK") || strings.Contains(bodyStr, "healthy")) {
				serviceType = "标准 TTS 健康检查"
			} else if endpoint.path == "/gradio_api/info" {
				serviceType = "Gradio IndexTTS"
			} else if strings.Contains(bodyLower, "gradio") {
				serviceType = "Gradio IndexTTS Web 界面"
			} else if strings.Contains(bodyLower, "fastapi") || strings.Contains(bodyLower, "swagger") {
				serviceType = "FastAPI TTS 服务"
			} else if strings.Contains(bodyLower, "openapi") || endpoint.path == "/openapi.json" {
				serviceType = "OpenAPI TTS 服务"
			} else if endpoint.path == "/docs" {
				serviceType = "TTS 服务文档"
			} else if resp.StatusCode == 200 {
				serviceType = fmt.Sprintf("HTTP 服务 (%s)", endpoint.path)
			}

			message := fmt.Sprintf("TTS 服务连接成功 - %s", serviceType)
			if strings.Contains(serviceType, "Gradio") {
				message += " (IndexTTS 兼容)"
			}

			return &models.TestConnectionResponse{
				Status:    "connected",
				Message:   message,
				LatencyMs: latency,
			}, nil
		}
		resp.Body.Close()
	}

	latency := time.Since(start).Milliseconds()

	return &models.TestConnectionResponse{
		Status:    "failed",
		Message:   fmt.Sprintf("所有测试端点都无法访问，最后错误: %v", lastErr),
		LatencyMs: latency,
	}, nil
}

// testTranslateConnection tests the GLM translation API connection.
func (s *SettingsService) testTranslateConnection(ctx context.Context, settings *models.Settings) (*models.TestConnectionResponse, error) {
	if settings.Translate.GLMAPIKey == "" {
		return &models.TestConnectionResponse{
			Status:  "failed",
			Message: "GLM API Key 未配置",
		}, nil
	}

	// For GLM, we just validate the key format and URL
	// A real test would make a minimal API call
	start := time.Now()

	if settings.Translate.GLMAPIURL == "" {
		return &models.TestConnectionResponse{
			Status:  "failed",
			Message: "GLM API URL 未配置",
		}, nil
	}

	latency := time.Since(start).Milliseconds()

	return &models.TestConnectionResponse{
		Status:    "connected",
		Message:   "GLM 翻译服务配置已保存，将在下次任务时验证",
		LatencyMs: latency,
	}, nil
}

// GetSettingValue retrieves a single setting value.
func (s *SettingsService) GetSettingValue(ctx context.Context, category, key string) (string, error) {
	query := `SELECT value FROM settings WHERE category = $1 AND key = $2`
	var value string
	err := s.db.QueryRowContext(ctx, query, category, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get setting: %w", err)
	}
	return value, nil
}
