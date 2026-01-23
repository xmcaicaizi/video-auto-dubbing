package handlers

import (
	"net/http"

	"vedio/api/internal/models"
	"vedio/api/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SettingsHandler handles settings-related requests.
type SettingsHandler struct {
	service *service.SettingsService
	logger  *zap.Logger
}

// NewSettingsHandler creates a new settings handler.
func NewSettingsHandler(service *service.SettingsService, logger *zap.Logger) *SettingsHandler {
	return &SettingsHandler{
		service: service,
		logger:  logger,
	}
}

// GetSettings handles GET /api/v1/settings.
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	settings, err := h.service.GetSettings(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get settings", zap.Error(err))
		h.respondError(c, http.StatusInternalServerError, 2001, "获取设置失败", err.Error())
		return
	}

	// Mask sensitive values before returning
	masked := settings.MaskSensitive()

	h.respondSuccess(c, masked)
}

// UpdateSettings handles PUT /api/v1/settings.
func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	var req models.SettingsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, 2002, "参数错误", err.Error())
		return
	}

	if err := h.service.UpdateSettings(c.Request.Context(), &req); err != nil {
		h.logger.Error("Failed to update settings", zap.Error(err))
		h.respondError(c, http.StatusInternalServerError, 2003, "保存设置失败", err.Error())
		return
	}

	h.respondSuccess(c, gin.H{"message": "设置已保存"})
}

// TestConnection handles POST /api/v1/settings/test.
func (h *SettingsHandler) TestConnection(c *gin.Context) {
	var req models.TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, 2004, "参数错误", err.Error())
		return
	}

	result, err := h.service.TestConnection(c.Request.Context(), req.Type)
	if err != nil {
		h.logger.Error("Failed to test connection", zap.Error(err))
		h.respondError(c, http.StatusInternalServerError, 2005, "测试连接失败", err.Error())
		return
	}

	h.respondSuccess(c, result)
}

// respondSuccess sends a success response.
func (h *SettingsHandler) respondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    data,
	})
}

// respondError sends an error response.
func (h *SettingsHandler) respondError(c *gin.Context, statusCode, code int, message, details string) {
	c.JSON(statusCode, gin.H{
		"code":    code,
		"message": message,
		"data":    details,
	})
}
