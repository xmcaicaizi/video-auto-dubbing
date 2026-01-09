package handlers

import (
	"net/http"
	"strconv"

	"vedio/api/internal/models"
	"vedio/api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TaskHandler handles task-related requests.
type TaskHandler struct {
	service *service.TaskService
	logger  *zap.Logger
}

// NewTaskHandler creates a new task handler.
func NewTaskHandler(service *service.TaskService, logger *zap.Logger) *TaskHandler {
	return &TaskHandler{
		service: service,
		logger:  logger,
	}
}

// CreateTaskRequest represents the request to create a task.
type CreateTaskRequest struct {
	SourceLanguage string `form:"source_language" binding:"omitempty"`
	TargetLanguage string `form:"target_language" binding:"omitempty"`
	// External credentials configured by user on frontend (optional)
	ASRAppID        string `form:"asr_appid" binding:"omitempty"`
	ASRToken        string `form:"asr_token" binding:"omitempty"`
	ASRCluster      string `form:"asr_cluster" binding:"omitempty"`
	ASRAPIKey       string `form:"asr_api_key" binding:"omitempty"`
	GLMAPIKey       string `form:"glm_api_key" binding:"omitempty"`
	GLMAPIURL       string `form:"glm_api_url" binding:"omitempty"`
	GLMModel        string `form:"glm_model" binding:"omitempty"`
	ModelScopeToken string `form:"modelscope_token" binding:"omitempty"`
}

// CreateTaskResponse represents the response for creating a task.
type CreateTaskResponse struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Data    *CreateTaskData     `json:"data"`
}

// CreateTaskData contains the task creation data.
type CreateTaskData struct {
	TaskID    string    `json:"task_id"`
	Status    string    `json:"status"`
	CreatedAt string    `json:"created_at"`
}

// CreateTask handles POST /api/v1/tasks.
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBind(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, 1001, "参数错误", err.Error())
		return
	}

	// Get uploaded file
	file, err := c.FormFile("video")
	if err != nil {
		h.respondError(c, http.StatusBadRequest, 1003, "文件上传失败", err.Error())
		return
	}

	// Set defaults
	if req.SourceLanguage == "" {
		req.SourceLanguage = "zh"
	}
	if req.TargetLanguage == "" {
		req.TargetLanguage = "en"
	}

	// Create task
	task, err := h.service.CreateTask(c.Request.Context(), file, req.SourceLanguage, req.TargetLanguage, service.CreateTaskOptions{
		ASRAppID:        req.ASRAppID,
		ASRToken:        req.ASRToken,
		ASRCluster:      req.ASRCluster,
		ASRAPIKey:       req.ASRAPIKey,
		GLMAPIKey:       req.GLMAPIKey,
		GLMAPIURL:       req.GLMAPIURL,
		GLMModel:        req.GLMModel,
		ModelScopeToken: req.ModelScopeToken,
	})
	if err != nil {
		h.logger.Error("Failed to create task", zap.Error(err))
		h.respondError(c, http.StatusInternalServerError, 1004, "内部服务错误", err.Error())
		return
	}

	h.respondSuccess(c, CreateTaskData{
		TaskID:    task.ID.String(),
		Status:    string(task.Status),
		CreatedAt: task.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// GetTask handles GET /api/v1/tasks/:task_id.
func (h *TaskHandler) GetTask(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("task_id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, 1001, "参数错误", "invalid task_id")
		return
	}

	task, steps, err := h.service.GetTaskWithSteps(c.Request.Context(), taskID)
	if err != nil {
		if err == service.ErrTaskNotFound {
			h.respondError(c, http.StatusNotFound, 1002, "任务不存在", "")
			return
		}
		h.logger.Error("Failed to get task", zap.Error(err))
		h.respondError(c, http.StatusInternalServerError, 1004, "内部服务错误", err.Error())
		return
	}

	// The DB status stays "queued" until the final step marks it "done"/"failed".
	// For API consumers (including the web UI), treat tasks with recorded steps as running.
	effectiveStatus := task.Status
	if task.Status == models.TaskStatusQueued && len(steps) > 0 {
		effectiveStatus = models.TaskStatusRunning
	}

	// Convert steps to response format
	stepResponses := make([]map[string]interface{}, len(steps))
	for i, step := range steps {
		stepResp := map[string]interface{}{
			"step":       step.Step,
			"status":     string(step.Status),
			"started_at": nil,
			"ended_at":  nil,
		}
		if step.StartedAt != nil {
			stepResp["started_at"] = step.StartedAt.Format("2006-01-02T15:04:05Z")
		}
		if step.EndedAt != nil {
			stepResp["ended_at"] = step.EndedAt.Format("2006-01-02T15:04:05Z")
		}
		stepResponses[i] = stepResp
	}

	h.respondSuccess(c, map[string]interface{}{
		"task_id":        task.ID.String(),
		"status":         string(effectiveStatus),
		"progress":       task.Progress,
		"source_language": task.SourceLanguage,
		"target_language": task.TargetLanguage,
		"error":          task.Error,
		"created_at":     task.CreatedAt.Format("2006-01-02T15:04:05Z"),
		"updated_at":     task.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		"steps":          stepResponses,
	})
}

// GetTaskResult handles GET /api/v1/tasks/:task_id/result.
func (h *TaskHandler) GetTaskResult(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("task_id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, 1001, "参数错误", "invalid task_id")
		return
	}

	result, err := h.service.GetTaskResult(c.Request.Context(), taskID)
	if err != nil {
		if err == service.ErrTaskNotFound {
			h.respondError(c, http.StatusNotFound, 1002, "任务不存在", "")
			return
		}
		if err == service.ErrTaskNotCompleted {
			h.respondError(c, http.StatusBadRequest, 1002, "任务未完成", "")
			return
		}
		h.logger.Error("Failed to get task result", zap.Error(err))
		h.respondError(c, http.StatusInternalServerError, 1004, "内部服务错误", err.Error())
		return
	}

	h.respondSuccess(c, result)
}

// GetTaskDownload handles GET /api/v1/tasks/:task_id/download.
func (h *TaskHandler) GetTaskDownload(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("task_id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, 1001, "参数错误", "invalid task_id")
		return
	}

	downloadType := c.DefaultQuery("type", "video")
	url, err := h.service.GetDownloadURL(c.Request.Context(), taskID, downloadType)
	if err != nil {
		if err == service.ErrTaskNotFound {
			h.respondError(c, http.StatusNotFound, 1002, "任务不存在", "")
			return
		}
		h.logger.Error("Failed to get download URL", zap.Error(err))
		h.respondError(c, http.StatusInternalServerError, 1004, "内部服务错误", err.Error())
		return
	}

	h.respondSuccess(c, map[string]interface{}{
		"download_url": url,
		"expires_in":   3600,
	})
}

// ListTasks handles GET /api/v1/tasks.
func (h *TaskHandler) ListTasks(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	tasks, total, err := h.service.ListTasks(c.Request.Context(), status, page, pageSize)
	if err != nil {
		h.logger.Error("Failed to list tasks", zap.Error(err))
		h.respondError(c, http.StatusInternalServerError, 1004, "内部服务错误", err.Error())
		return
	}

	taskList := make([]map[string]interface{}, len(tasks))
	for i, task := range tasks {
		taskList[i] = map[string]interface{}{
			"task_id":    task.ID.String(),
			"status":     string(task.Status),
			"progress":    task.Progress,
			"created_at": task.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	h.respondSuccess(c, map[string]interface{}{
		"tasks":     taskList,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// DeleteTask handles DELETE /api/v1/tasks/:task_id.
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("task_id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, 1001, "参数错误", "invalid task_id")
		return
	}

	if err := h.service.DeleteTask(c.Request.Context(), taskID); err != nil {
		if err == service.ErrTaskNotFound {
			h.respondError(c, http.StatusNotFound, 1002, "任务不存在", "")
			return
		}
		h.logger.Error("Failed to delete task", zap.Error(err))
		h.respondError(c, http.StatusInternalServerError, 1004, "内部服务错误", err.Error())
		return
	}

	h.respondSuccess(c, nil)
}

// respondSuccess sends a success response.
func (h *TaskHandler) respondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    data,
	})
}

// respondError sends an error response.
func (h *TaskHandler) respondError(c *gin.Context, statusCode, code int, message, details string) {
	c.JSON(statusCode, gin.H{
		"code":    code,
		"message": message,
		"data":    details,
	})
}

