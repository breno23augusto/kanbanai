package http

import (
	"os"
	"path/filepath"
	"strings"

	"kanbanai/internal/adapter/in/http/handler"
	"kanbanai/internal/adapter/in/http/middleware"
	"kanbanai/internal/adapter/in/http/response"
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
		api.POST("/tasks/:id/complete", taskHandler.CompletePhase)

		sseHandler := container.MustResolve("sseHandler").(*handler.SSEHandler)
		api.GET("/events", sseHandler.Stream)
	}

	mountFrontend(r, webDir)
}

// mountFrontend serves the built React frontend (SPEC §16.4). When webDir does
// not contain an index.html, frontend serving is skipped so the API still works
// in headless deployments.
func mountFrontend(r *gin.Engine, webDir string) {
	if webDir == "" {
		return
	}
	indexFile := filepath.Join(webDir, "index.html")
	if _, err := os.Stat(indexFile); err != nil {
		return
	}

	r.Static("/assets", filepath.Join(webDir, "assets"))
	r.StaticFile("/", indexFile)

	// SPA fallback: serve index.html for unknown non-API routes so client-side
	// routing works. API 404s still return a structured JSON error.
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") || strings.HasPrefix(c.Request.URL.Path, "/mcp/") {
			response.NotFound(c, "resource not found")
			return
		}
		c.File(indexFile)
	})
}