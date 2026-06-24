package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kanbanai/config"
	"kanbanai/internal/adapter/in/mcp"
	"kanbanai/internal/di"

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

	apiSrv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: r,
	}

	// MCP runs on a dedicated port (SPEC §13.1). The harness connects to
	// http://localhost:<mcpPort>/mcp/sse as injected by the CommandBuilder.
	mcpMux := http.NewServeMux()
	mcpMux.Handle("/mcp/", mcp.NewMCPHandler(container))
	mcpSrv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.MCP.Port),
		Handler: mcpMux,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	apiErr := make(chan error, 1)
	mcpErr := make(chan error, 1)

	go func() {
		slog.Info("HTTP server starting", "addr", apiSrv.Addr)
		if err := apiSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			apiErr <- err
		}
	}()

	go func() {
		slog.Info("MCP server starting", "addr", mcpSrv.Addr)
		if err := mcpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			mcpErr <- err
		}
	}()

	select {
	case err := <-apiErr:
		_ = mcpSrv.Close()
		return fmt.Errorf("http server: %w", err)
	case err := <-mcpErr:
		_ = apiSrv.Close()
		return fmt.Errorf("mcp server: %w", err)
	case <-quit:
	}

	slog.Info("Shutting down servers...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var shutdownErr error
	if err := apiSrv.Shutdown(shutdownCtx); err != nil {
		shutdownErr = fmt.Errorf("http shutdown: %w", err)
	}
	// The SSE handler drains connections within the shutdown deadline; the
	// 5s context above bounds the wait (SPEC §29).
	if err := mcpSrv.Shutdown(shutdownCtx); err != nil {
		shutdownErr = fmt.Errorf("mcp shutdown: %w", err)
	}

	slog.Info("Servers stopped")
	return shutdownErr
}