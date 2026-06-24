package config

import (
	"fmt"
	"kanbanai/internal/domain/entity"
	"os"

	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")

	v.AutomaticEnv()

	// Defaults
	v.SetDefault("KANBANAI_SERVER_PORT", 8080)
	v.SetDefault("KANBANAI_SERVER_HOST", "0.0.0.0")
	v.SetDefault("KANBANAI_DB_PATH", "./data/kanbanai.db")
	v.SetDefault("KANBANAI_DB_MIGRATION_DIR", "./internal/adapter/out/persistence/sqlite/migration_files")
	v.SetDefault("KANBANAI_MCP_PORT", 8081)
	v.SetDefault("KANBANAI_HARNESS_DEFAULT_CMD", "claude")
	v.SetDefault("KANBANAI_HARNESS_DEFAULT_MODEL", "claude-sonnet-4-20250514")
	v.SetDefault("KANBANAI_HARNESS_MAX_RETRIES", 3)
	v.SetDefault("KANBANAI_HARNESS_TIMEOUT_SEC", 600)
	v.SetDefault("KANBANAI_WEB_DIR", "./web")
	v.SetDefault("KANBANAI_LOG_LEVEL", "info")

	if err := v.ReadInConfig(); err != nil {
		// A missing .env is fine — defaults + env vars are enough.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("read config: %w", err)
			}
		}
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: v.GetInt("KANBANAI_SERVER_PORT"),
			Host: v.GetString("KANBANAI_SERVER_HOST"),
		},
		DB: DBConfig{
			Path:         v.GetString("KANBANAI_DB_PATH"),
			MigrationDir: v.GetString("KANBANAI_DB_MIGRATION_DIR"),
		},
		MCP: MCPConfig{
			Port: v.GetInt("KANBANAI_MCP_PORT"),
		},
		Harness: HarnessConfig{
			DefaultCmd:        v.GetString("KANBANAI_HARNESS_DEFAULT_CMD"),
			DefaultModel:      v.GetString("KANBANAI_HARNESS_DEFAULT_MODEL"),
			DefaultMaxRetries: v.GetInt("KANBANAI_HARNESS_MAX_RETRIES"),
			DefaultTimeoutSec: v.GetInt("KANBANAI_HARNESS_TIMEOUT_SEC"),
			Phases:            loadPhaseConfigs(v),
		},
		Web: WebConfig{
			Dir: v.GetString("KANBANAI_WEB_DIR"),
		},
		Log: LogConfig{
			Level: v.GetString("KANBANAI_LOG_LEVEL"),
		},
	}

	return cfg, nil
}

func loadPhaseConfigs(v *viper.Viper) map[entity.Phase]PhaseHarnessConfig {
	phases := make(map[entity.Phase]PhaseHarnessConfig)

	phaseConfigs := map[entity.Phase]struct {
		cmdKey    string
		modelKey  string
		retryKey  string
		timeoutKey string
	}{
		entity.PhasePlanning: {
			cmdKey:    "KANBANAI_HARNESS_PLANNING_CMD",
			modelKey:  "KANBANAI_HARNESS_PLANNING_MODEL",
			retryKey:  "KANBANAI_HARNESS_PLANNING_MAX_RETRIES",
			timeoutKey: "KANBANAI_HARNESS_PLANNING_TIMEOUT_SEC",
		},
		entity.PhaseTodo: {
			cmdKey:    "KANBANAI_HARNESS_TODO_CMD",
			modelKey:  "KANBANAI_HARNESS_TODO_MODEL",
			retryKey:  "KANBANAI_HARNESS_TODO_MAX_RETRIES",
			timeoutKey: "KANBANAI_HARNESS_TODO_TIMEOUT_SEC",
		},
		entity.PhaseDoing: {
			cmdKey:    "KANBANAI_HARNESS_DOING_CMD",
			modelKey:  "KANBANAI_HARNESS_DOING_MODEL",
			retryKey:  "KANBANAI_HARNESS_DOING_MAX_RETRIES",
			timeoutKey: "KANBANAI_HARNESS_DOING_TIMEOUT_SEC",
		},
		entity.PhaseValidating: {
			cmdKey:    "KANBANAI_HARNESS_VALIDATING_CMD",
			modelKey:  "KANBANAI_HARNESS_VALIDATING_MODEL",
			retryKey:  "KANBANAI_HARNESS_VALIDATING_MAX_RETRIES",
			timeoutKey: "KANBANAI_HARNESS_VALIDATING_TIMEOUT_SEC",
		},
		entity.PhaseTesting: {
			cmdKey:    "KANBANAI_HARNESS_TESTING_CMD",
			modelKey:  "KANBANAI_HARNESS_TESTING_MODEL",
			retryKey:  "KANBANAI_HARNESS_TESTING_MAX_RETRIES",
			timeoutKey: "KANBANAI_HARNESS_TESTING_TIMEOUT_SEC",
		},
	}

	for phase, keys := range phaseConfigs {
		phases[phase] = PhaseHarnessConfig{
			Cmd:        v.GetString(keys.cmdKey),
			Model:      v.GetString(keys.modelKey),
			MaxRetries: v.GetInt(keys.retryKey),
			TimeoutSec: v.GetInt(keys.timeoutKey),
		}
	}

	return phases
}
