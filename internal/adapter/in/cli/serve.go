package cli

import (
	"kanbanai/config"
	"kanbanai/internal/adapter/bootstrap"
	"kanbanai/internal/adapter/in/http"
	"log/slog"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP + MCP servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		container, err := bootstrap.Initialize(cfg)
		if err != nil {
			return err
		}

		slog.Info("KanbanAI server starting",
			"http_port", cfg.Server.Port,
			"mcp_port", cfg.MCP.Port,
			"db_path", cfg.DB.Path,
		)

		return http.StartServer(cfg, container)
	},
}
