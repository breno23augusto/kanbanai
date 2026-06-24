# KanbanAI — Configuração e CLI

## 1. Variáveis de Ambiente

```env
# .env.example

# Servidor
KANBANAI_SERVER_PORT=8080
KANBANAI_SERVER_HOST=0.0.0.0

# Banco de Dados
KANBANAI_DB_PATH=./data/kanbanai.db

# MCP
KANBANAI_MCP_PORT=8081

# Harness Padrão
KANBANAI_HARNESS_DEFAULT_CMD=claude
KANBANAI_HARNESS_DEFAULT_MODEL=claude-sonnet-4-20250514

# Modelos por Raia (override do default)
KANBANAI_HARNESS_PLANNING_CMD=claude
KANBANAI_HARNESS_PLANNING_MODEL=claude-sonnet-4-20250514
KANBANAI_HARNESS_TODO_CMD=claude
KANBANAI_HARNESS_TODO_MODEL=claude-sonnet-4-20250514
KANBANAI_HARNESS_DOING_CMD=claude
KANBANAI_HARNESS_DOING_MODEL=claude-sonnet-4-20250514
KANBANAI_HARNESS_VALIDATING_CMD=claude
KANBANAI_HARNESS_VALIDATING_MODEL=claude-sonnet-4-20250514
KANBANAI_HARNESS_TESTING_CMD=claude
KANBANAI_HARNESS_TESTING_MODEL=claude-sonnet-4-20250514

# Retry e Timeout (global, sobrescrito por raia)
KANBANAI_HARNESS_MAX_RETRIES=3
KANBANAI_HARNESS_TIMEOUT_SEC=600
KANBANAI_HARNESS_PLANNING_MAX_RETRIES=3
KANBANAI_HARNESS_PLANNING_TIMEOUT_SEC=600
KANBANAI_HARNESS_TODO_MAX_RETRIES=3
KANBANAI_HARNESS_TODO_TIMEOUT_SEC=600
KANBANAI_HARNESS_DOING_MAX_RETRIES=3
KANBANAI_HARNESS_DOING_TIMEOUT_SEC=900
KANBANAI_HARNESS_VALIDATING_MAX_RETRIES=3
KANBANAI_HARNESS_VALIDATING_TIMEOUT_SEC=600
KANBANAI_HARNESS_TESTING_MAX_RETRIES=3
KANBANAI_HARNESS_TESTING_TIMEOUT_SEC=900

# Frontend
KANBANAI_FRONTEND_URL=http://localhost:3000

# Log Level
KANBANAI_LOG_LEVEL=info
```

---

## 2. Struct de Configuração

```go
// config/config.go
type Config struct {
    Server  ServerConfig
    DB      DBConfig
    MCP     MCPConfig
    Harness HarnessConfig
    Web     WebConfig
    Log     LogConfig
}

type MCPConfig struct {
    Port int // Porta do servidor MCP (default: 8081)
}

type WebConfig struct {
    Dir string // Diretório dos arquivos estáticos do frontend (default: ./web)
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
    MaxRetries int // Sobrescreve DefaultMaxRetries
    TimeoutSec int // Sobrescreve DefaultTimeoutSec
}
```

---

## 3. CLI (Cobra)

```go
// internal/adapter/in/cli/root.go
var rootCmd = &cobra.Command{
    Use:   "kanbanai",
    Short: "KanbanAI — AI-powered Kanban orchestrator",
}

// internal/adapter/in/cli/serve.go
var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start the HTTP + MCP servers",
    RunE:  runServe,
}

// internal/adapter/in/cli/migrate.go
var migrateCmd = &cobra.Command{
    Use:   "migrate",
    Short: "Run database migrations",
    RunE:  runMigrate,
}
```

### Comandos Disponíveis

| Comando     | Descrição                        |
|-------------|----------------------------------|
| `kanbanai serve`   | Inicia servidores HTTP + MCP |
| `kanbanai migrate` | Executa migrations do banco  |
| `kanbanai version` | Mostra versão da aplicação   |
