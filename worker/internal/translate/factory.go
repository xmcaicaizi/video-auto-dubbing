package translate

import (
	"fmt"

	"vedio/worker/internal/config"

	"go.uber.org/zap"
)

// ProviderType represents the translation service provider.
type ProviderType string

const (
	ProviderGLM       ProviderType = "glm"       // Zhipu GLM (智谱AI)
	ProviderDashScope ProviderType = "dashscope" // Aliyun DashScope (阿里百炼)
)

// NewTranslator creates a new translator based on the provider type.
// If provider is empty or invalid, defaults to GLM for backward compatibility.
func NewTranslator(provider ProviderType, effectiveConfig *config.EffectiveConfig, logger *zap.Logger) (Translator, error) {
	switch provider {
	case ProviderDashScope:
		if effectiveConfig.External.DashScope.APIKey == "" {
			return nil, fmt.Errorf("DASHSCOPE_LLM_API_KEY is required for DashScope provider")
		}
		logger.Info("Creating DashScope translator",
			zap.String("model", effectiveConfig.External.DashScope.Model),
			zap.String("base_url", effectiveConfig.External.DashScope.BaseURL),
		)
		return NewDashScopeClient(effectiveConfig.External.DashScope, logger), nil

	case ProviderGLM, "":
		// Default to GLM for backward compatibility
		if effectiveConfig.External.GLM.APIKey == "" {
			return nil, fmt.Errorf("GLM_API_KEY is required for GLM provider")
		}
		logger.Info("Creating GLM translator",
			zap.String("model", effectiveConfig.External.GLM.Model),
			zap.String("api_url", effectiveConfig.External.GLM.APIURL),
		)
		return NewClient(effectiveConfig.External.GLM, logger), nil

	default:
		return nil, fmt.Errorf("unsupported translation provider: %s", provider)
	}
}
