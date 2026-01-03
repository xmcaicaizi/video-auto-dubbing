package router

import (
	"time"

	"vedio/api/internal/database"
	"vedio/api/internal/handlers"
	"vedio/api/internal/orchestrator"
	"vedio/api/internal/queue"
	"vedio/api/internal/service"
	"vedio/api/internal/storage"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// New creates a new router with all routes configured.
func New(db *database.DB, storage *storage.Service, publisher *queue.Publisher, logger *zap.Logger) *gin.Engine {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// Middleware
	r.Use(ginLogger(logger))
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Initialize services
		taskRepo := orchestrator.NewDBTaskRepository(db)
		taskOrchestrator := orchestrator.NewTaskOrchestrator(publisher, taskRepo)
		taskService := service.NewTaskService(db, storage, taskOrchestrator)
		taskHandler := handlers.NewTaskHandler(taskService, logger)

		// Task routes
		tasks := v1.Group("/tasks")
		{
			tasks.POST("", taskHandler.CreateTask)
			tasks.GET("", taskHandler.ListTasks)
			tasks.GET("/:task_id", taskHandler.GetTask)
			tasks.GET("/:task_id/result", taskHandler.GetTaskResult)
			tasks.GET("/:task_id/download", taskHandler.GetTaskDownload)
			tasks.DELETE("/:task_id", taskHandler.DeleteTask)
		}
	}

	return r
}

// ginLogger is a custom logger middleware.
func ginLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		logger.Info("HTTP request",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
		)
	}
}

// corsMiddleware adds CORS headers.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
