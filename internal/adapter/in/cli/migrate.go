package cli

import (
	"fmt"
	"kanbanai/config"
	"kanbanai/internal/adapter/out/persistence/sqlite"
	"log/slog"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		db, err := sqlite.NewConnection(cfg.DB.Path)
		if err != nil {
			return fmt.Errorf("connect to database: %w", err)
		}
		defer db.Close()

		if err := sqlite.RunMigrations(db, cfg.DB.MigrationDir); err != nil {
			return fmt.Errorf("run migrations: %w", err)
		}

		slog.Info("Migrations completed successfully")
		return nil
	},
}
