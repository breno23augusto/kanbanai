package config

import "kanbanai/internal/domain/entity"

type Config struct {
	Server  ServerConfig
	DB      DBConfig
	MCP     MCPConfig
	Harness HarnessConfig
	Web     WebConfig
	Log     LogConfig
}

type ServerConfig struct {
	Port int
	Host string
}

type DBConfig struct {
	Path         string
	MigrationDir string
}

type MCPConfig struct {
	Port int
}

type HarnessConfig struct {
	DefaultCmd        string
	DefaultModel      string
	DefaultMaxRetries int
	DefaultTimeoutSec int
	Phases            map[entity.Phase]PhaseHarnessConfig
}

type PhaseHarnessConfig struct {
	Cmd        string
	Model      string
	MaxRetries int
	TimeoutSec int
}

type WebConfig struct {
	Dir string
}

type LogConfig struct {
	Level string
}
