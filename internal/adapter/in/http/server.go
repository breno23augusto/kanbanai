package http

import (
	"context"
	"fmt"
	"kanbanai/config"
	"kanbanai/internal/adapter/in/mcp"
	"kanbanai/internal/di"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func StartServer(cfg *config.Config, container *di.Container) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	webDir := cfg.Web.Dir
	if webDir == "" {
		webDir = "./web"
	}

	SetupRoutes(r, container, webDir)

	// Mount MCP SSE handler
	mcpHandler := mcp.NewMCPHandler(container)
	r.Any("/mcp/*path", gin.WrapH(mcpHandler))

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: r,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("HTTP server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	<-quit
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	slog.Info("Server stopped")
	return nil
}
