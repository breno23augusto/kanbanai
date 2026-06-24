package http

import (
	"kanbanai/internal/adapter/in/http/handler"
	"kanbanai/internal/adapter/in/http/middleware"
	"kanbanai/internal/di"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, container *di.Container, webDir string) {
	r.Use(middleware.CORS())
	r.Use(middleware.RequestID())
	r.Use(middleware.ErrorHandler())

	api := r.Group("/api/v1")
	{
		healthHandler := container.MustResolve("healthHandler").(*handler.HealthHandler)
		api.GET("/health", healthHandler.Check)

		taskHandler := container.MustResolve("taskHandler").(*handler.TaskHandler)
		api.POST("/tasks", taskHandler.Create)
		api.GET("/tasks", taskHandler.List)
		api.GET("/tasks/:id", taskHandler.Get)
		api.PUT("/tasks/:id", taskHandler.Update)
		api.DELETE("/tasks/:id", taskHandler.Delete)
		api.GET("/tasks/:id/timeline", taskHandler.GetTimeline)
		api.POST("/tasks/:id/retry", taskHandler.Retry)

		sseHandler := container.MustResolve("sseHandler").(*handler.SSEHandler)
		api.GET("/events", sseHandler.Stream)
	}
}
