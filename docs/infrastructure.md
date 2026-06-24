# KanbanAI — Infraestrutura e Adapters

## 1. Estrutura de Diretórios

```
kanbanai/
├── cmd/
│   └── kanbanai/
│       └── main.go                          # Entrypoint
│
├── internal/
│   ├── di/
│   │   ├── container.go                     # Container DI genérico
│   │   └── container_test.go
│   │
│   ├── domain/
│   │   ├── entity/
│   │   │   ├── task.go                      # Entidade Task
│   │   │   ├── task_phase.go                # Enum de fases
│   │   │   ├── task_status.go               # Enum de status
│   │   │   ├── task_event_log.go            # Entidade de log de eventos
│   │   │   ├── phase_config.go              # Config de modelo por raia
│   │   │   └── phase_output.go              # Entidade PhaseOutput
│   │   │
│   │   ├── event/
│   │   │   ├── types.go                     # Definição dos EventTypes
│   │   │   ├── event.go                     # Struct Event
│   │   │   ├── dispatcher.go                # Interface Dispatcher
│   │   │   └── handler.go                   # Type Handler
│   │   │
│   │   ├── repository/
│   │   │   ├── task_repository.go           # Interface TaskRepository
│   │   │   ├── task_event_log_repository.go # Interface TaskEventLogRepository
│   │   │   └── phase_output_repository.go   # Interface PhaseOutputRepository
│   │   │
│   │   ├── query/
│   │   │   ├── task_with_phases_query.go    # Interface — join task + phases
│   │   │   └── task_timeline_query.go       # Interface — join task + events
│   │   │
│   │   └── port/
│   │       ├── harness_port.go              # Interface de saída para harness
│   │       ├── sse_port.go                  # Interface de saída para SSE
│   │       └── phase_orchestrator_port.go   # Interface do orquestrador
│   │
│   ├── application/
│   │   ├── usecase/
│   │   │   ├── create_task.go               # UseCase: criar task
│   │   │   ├── create_task_test.go
│   │   │   ├── update_task.go               # UseCase: atualizar task
│   │   │   ├── update_task_test.go
│   │   │   ├── delete_task.go               # UseCase: deletar task
│   │   │   ├── delete_task_test.go
│   │   │   ├── get_task.go                  # UseCase: buscar task
│   │   │   ├── get_task_test.go
│   │   │   ├── list_tasks.go                # UseCase: listar tasks com filtros
│   │   │   ├── list_tasks_test.go
│   │   │   ├── advance_phase.go             # UseCase: avançar fase da task
│   │   │   ├── advance_phase_test.go
│   │   │   ├── report_phase_progress.go     # UseCase: reportar progresso
│   │   │   └── report_phase_progress_test.go
│   │   │
│   │   ├── dto/
│   │   │   ├── create_task_input.go         # DTO de entrada
│   │   │   ├── task_output.go               # DTO de saída
│   │   │   ├── task_filter.go               # DTO de filtros
│   │   │   └── phase_progress.go            # DTO de progresso
│   │   │
│   │   └── service/
│   │       ├── phase_orchestrator.go         # Orquestrador de fases
│   │       ├── phase_orchestrator_test.go
│   │       ├── prompt_builder.go             # Construtor de prompts por fase
│   │       └── prompt_builder_test.go
│   │
│   └── adapter/
│       ├── bootstrap/
│       │   └── bootstrap.go                 # Setup do DI e registro de event subscribers
│       │
│       ├── in/
│       │   ├── http/
│       │   │   ├── server.go                # Setup do Gin + middleware
│       │   │   ├── router.go                # Definição de rotas
│       │   │   ├── middleware/
│       │   │   │   ├── cors.go              # CORS middleware
│       │   │   │   ├── error_handler.go     # Error handler global
│       │   │   │   └── request_id.go        # Request ID middleware
│       │   │   ├── handler/
│       │   │   │   ├── task_handler.go       # Handlers de task
│       │   │   │   ├── task_handler_test.go
│       │   │   │   ├── sse_handler.go        # Handler SSE
│       │   │   │   ├── sse_handler_test.go
│       │   │   │   └── health_handler.go     # Handler health check
│       │   │   └── response/
│       │   │       ├── success.go            # Response padronizado ok
│       │   │       └── error.go              # Response padronizado erro
│       │   │
│       │   ├── cli/
│       │   │   ├── root.go                   # Cobra root command
│       │   │   ├── serve.go                  # Cobra: start server
│       │   │   ├── migrate.go                # Cobra: run migrations
│       │   │   └── version.go                # Cobra: show version
│       │   │
│       │   └── mcp/
│       │       ├── server.go                 # MCP Server setup
│       │       ├── tools.go                  # Registro de tools MCP
│       │       ├── tool_report_progress.go   # Tool: report_progress
│       │       ├── tool_update_output.go     # Tool: update_task_output
│       │       ├── tool_complete_phase.go    # Tool: complete_phase
│       │       └── tool_get_task.go          # Tool: get_task
│       │
│       └── out/
│           ├── persistence/
│           │   ├── sqlite/
│           │   │   ├── connection.go         # SQLite connection manager
│           │   │   ├── migration.go          # Schema migrations
│           │   │   └── migration_files/
│           │   │       ├── 001_create_tasks.sql
│           │   │       ├── 002_create_task_event_logs.sql
│           │   │       └── 003_create_phase_outputs.sql
│           │   │
│           │   ├── repository/
│           │   │   ├── task_repository_sqlite.go
│           │   │   ├── task_repository_sqlite_test.go
│           │   │   ├── task_event_log_repository_sqlite.go
│           │   │   ├── task_event_log_repository_sqlite_test.go
│           │   │   ├── phase_output_repository_sqlite.go
│           │   │   └── phase_output_repository_sqlite_test.go
│           │   │
│           │   └── query/
│           │       ├── task_with_phases_query_sqlite.go
│           │       ├── task_with_phases_query_sqlite_test.go
│           │       ├── task_timeline_query_sqlite.go
│           │       └── task_timeline_query_sqlite_test.go
│           │
│           ├── harness/
│           │   ├── adapter.go                # HarnessAdapter principal
│           │   ├── adapter_test.go
│           │   ├── command_builder.go        # Builder do comando CLI
│           │   ├── command_builder_test.go
│           │   └── config.go                 # Config de modelos por raia
│           │
│           ├── event/
│           │   ├── dispatcher_memory.go      # Impl in-memory do Dispatcher
│           │   └── dispatcher_memory_test.go
│           │
│           └── sse/
│               ├── broker.go                 # SSE Broker (hub de conexões)
│               ├── broker_test.go
│               ├── client.go                 # SSE Client connection
│               └── formatter.go              # Formata Event → SSE string
│
├── pkg/
│   ├── uid/
│   │   └── generator.go                     # Gerador de IDs (ULID/UUID)
│   └── clock/
│       └── clock.go                         # Abstração de time.Now()
│
├── config/
│   ├── config.go                            # Struct de configuração
│   ├── loader.go                            # Carregamento via Viper
│   └── defaults.go                          # Valores padrão
│
├── frontend/
│   ├── package.json
│   ├── src/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   ├── theme/
│   │   │   └── theme.ts                     # Tema MUI customizado
│   │   ├── hooks/
│   │   │   ├── useSSE.ts                    # Hook para SSE
│   │   │   └── useTasks.ts                  # Hook para tasks
│   │   ├── services/
│   │   │   └── api.ts                       # Client HTTP
│   │   ├── components/
│   │   │   ├── KanbanBoard.tsx              # Board principal
│   │   │   ├── KanbanLane.tsx               # Raia individual
│   │   │   ├── TaskCard.tsx                 # Card da task
│   │   │   ├── CreateTaskDialog.tsx         # Modal de criação
│   │   │   ├── TaskDetailDrawer.tsx         # Drawer de detalhes
│   │   │   ├── EventTimeline.tsx            # Timeline de eventos
│   │   │   └── PhaseProgress.tsx            # Progresso da fase
│   │   ├── pages/
│   │   │   └── Dashboard.tsx                # Página principal
│   │   └── types/
│   │       ├── task.ts                      # Tipos de task
│   │       └── event.ts                     # Tipos de evento
│   └── public/
│       └── index.html
│
├── .env.example
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
├── README.md
└── SPEC.md
```

---

## 2. MCP Server — Tools

O servidor MCP expõe tools para que o harness interaja com o sistema:

### 2.1 Tools Disponíveis

| Tool                    | Método     | Descrição                                            |
|-------------------------|------------|------------------------------------------------------|
| `get_task`              | Read       | Busca informações da task atual                      |
| `report_progress`       | Write      | Reporta progresso parcial da fase em execução        |
| `update_task_output`    | Write      | Salva artefatos/outputs da fase (plano, código etc.) |
| `complete_phase`        | Write      | Marca fase como concluída, dispara próxima           |

### 2.2 Implementação

```go
// internal/adapter/in/mcp/tools.go
func RegisterTools(server *mcp.Server, container *di.Container) {
    server.AddTool(reportProgressTool(container))
    server.AddTool(updateTaskOutputTool(container))
    server.AddTool(completePhaseTool(container))
    server.AddTool(getTaskTool(container))
}
```

Cada tool é definida em seu próprio arquivo:

```go
// internal/adapter/in/mcp/tool_complete_phase.go
func completePhaseTool(container *di.Container) mcp.Tool {
    return mcp.Tool{
        Name:        "complete_phase",
        Description: "Marks the current phase as completed. The next phase is started automatically by the orchestrator.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "task_id": {Type: "string", Description: "ID of the task"},
                "phase":   {Type: "string", Description: "Current phase being completed"},
                "summary": {Type: "string", Description: "Summary of what was accomplished"},
            },
            Required: []string{"task_id", "phase", "summary"},
        },
        Handler: func(ctx context.Context, args map[string]any) (any, error) {
            advancePhase := container.MustResolve("advancePhaseUseCase").(*usecase.AdvancePhase)
            return advancePhase.Execute(ctx, args["task_id"].(string))
        },
    }
}

// internal/adapter/in/mcp/tool_update_output.go
func updateTaskOutputTool(container *di.Container) mcp.Tool {
    return mcp.Tool{
        Name:        "update_task_output",
        Description: "Saves artifacts/outputs for the current phase (plan, code, test results, etc.)",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "task_id": {Type: "string", Description: "ID of the task"},
                "phase":   {Type: "string", Description: "Current phase"},
                "output":  {Type: "string", Description: "Raw output content (markdown, code, JSON, etc.)"},
                "summary": {Type: "string", Description: "Human-readable summary of the output"},
            },
            Required: []string{"task_id", "phase", "output"},
        },
        Handler: func(ctx context.Context, args map[string]any) (any, error) {
            savePhaseOutput := container.MustResolve("savePhaseOutputUseCase").(*usecase.SavePhaseOutput)
            input := dto.SavePhaseOutputInput{
                TaskID:  args["task_id"].(string),
                Phase:   entity.Phase(args["phase"].(string)),
                Output:  args["output"].(string),
                Summary: args["summary"].(string),
            }
            return savePhaseOutput.Execute(ctx, input)
        },
    }
}

// internal/adapter/in/mcp/tool_get_task.go
func getTaskTool(container *di.Container) mcp.Tool {
    return mcp.Tool{
        Name:        "get_task",
        Description: "Retrieves the current task information including phase outputs",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "task_id": {Type: "string", Description: "ID of the task to retrieve"},
            },
            Required: []string{"task_id"},
        },
        Handler: func(ctx context.Context, args map[string]any) (any, error) {
            getTask := container.MustResolve("getTaskUseCase").(*usecase.GetTask)
            return getTask.Execute(ctx, args["task_id"].(string))
        },
    }
}
```

---

## 3. SSE (Server-Sent Events)

### 3.1 Broker

```go
// internal/adapter/out/sse/broker.go
type Broker struct {
    mu          sync.RWMutex
    clients     map[string]chan event.Event
    dispatcher  event.Dispatcher
}

func NewBroker(dispatcher event.Dispatcher) *Broker {
    b := &Broker{
        clients:    make(map[string]chan event.Event),
        dispatcher: dispatcher,
    }
    // Subscribe to ALL events via wildcard and forward to connected clients
    dispatcher.SubscribeAll(b.onEvent)
    return b
}

// onEvent is the wildcard handler that forwards every event to all connected clients
func (b *Broker) onEvent(evt event.Event) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    for _, ch := range b.clients {
        select {
        case ch <- evt:
        default:
            // Client buffer full — drop event to avoid blocking the publisher
        }
    }
}

func (b *Broker) Subscribe(clientID string) <-chan event.Event {
    b.mu.Lock()
    defer b.mu.Unlock()
    ch := make(chan event.Event, 64) // Buffered to avoid slow-client head-of-line blocking
    b.clients[clientID] = ch
    return ch
}

func (b *Broker) Unsubscribe(clientID string) {
    b.mu.Lock()
    defer b.mu.Unlock()
    if ch, ok := b.clients[clientID]; ok {
        close(ch)
        delete(b.clients, clientID)
    }
}
```

### 3.2 Handler HTTP

```go
// internal/adapter/in/http/handler/sse_handler.go
func (h *SSEHandler) Stream(c *gin.Context) {
    clientID := uid.New()
    ch := h.broker.Subscribe(clientID)
    defer h.broker.Unsubscribe(clientID)

    c.Stream(func(w io.Writer) bool {
        select {
        case evt := <-ch:
            c.SSEvent(string(evt.Type), evt.Payload)
            return true
        case <-c.Request.Context().Done():
            return false
        }
    })
}
```

---

## 4. Harness Adapter

### 4.1 Protocolo de Transporte MCP e Descoberta

O harness conecta-se ao servidor MCP utilizando um dos dois transportes suportados:

1. **Stdio Transport (Padrão para CLI local)**: O `HarnessAdapter` spawna o executável do harness como processo filho. O harness e o KanbanAI se comunicam diretamente via `stdin` e `stdout` redirecionados do processo filho.
2. **SSE Transport (HTTP)**: O servidor MCP abre uma porta HTTP dedicada (`KANBANAI_MCP_PORT=8081`). O `HarnessAdapter` injeta a URL do endpoint SSE (`http://localhost:8081/mcp/sse`) nas variáveis de ambiente do processo filho (`KANBANAI_MCP_URL`).

### 4.2 Fluxo de Execução com Retries e Timeout

Se o harness falhar em responder dentro do limite definido por `TimeoutSec`, ou retornar um código de erro diferente de zero, o `PhaseOrchestrator` intercepta o erro através do monitoramento do processo e inicia a política de **Retry**:

1. **Backoff Linear**: Um intervalo de espera curto (`2 * tentativa` segundos) é observado antes de re-despachar o comando.
2. **Controle de Tentativas**: O orchestrator incrementa a contagem de tentativas da fase. Se `tentativas > MaxRetries`, o status da task é alterado para `StatusFailed` e o evento `phase.<phase>.failed` é publicado.
3. **Eventos de Retry**: Cada falha transitória dispara um evento `phase.<phase>.retry` para atualizar o frontend via SSE.

### 4.3 Implementação do Adapter

```go
// internal/adapter/out/harness/adapter.go
type Adapter struct {
    configs         map[entity.Phase]entity.PhaseConfig
    builder         *CommandBuilder
    dispatcher      event.Dispatcher
    processRegistry map[string]*exec.Cmd  // taskID → running harness process
    mu              sync.RWMutex
}

func (a *Adapter) RegisterProcess(taskID string, cmd *exec.Cmd) {
    a.mu.Lock()
    a.processRegistry[taskID] = cmd
    a.mu.Unlock()
}

func (a *Adapter) UnregisterProcess(taskID string) {
    a.mu.Lock()
    delete(a.processRegistry, taskID)
    a.mu.Unlock()
}

func (a *Adapter) GetProcess(taskID string) *exec.Cmd {
    a.mu.RLock()
    defer a.mu.RUnlock()
    return a.processRegistry[taskID]
}

func (a *Adapter) Dispatch(ctx context.Context, task *entity.Task, phase entity.Phase, prompt string) error {
    config := a.configs[phase]

    a.dispatcher.Publish(event.Event{
        Type:    event.HarnessCommandDispatched,
        TaskID:  task.ID,
        Payload: map[string]any{"phase": phase, "model": config.ModelName},
    })

    cmd, err := a.builder.Build(ctx, config, task.ID, prompt)
    if err != nil {
        return fmt.Errorf("failed to build command: %w", err)
    }

    if err := cmd.Start(); err != nil {
        return fmt.Errorf("failed to start harness: %w", err)
    }

    a.RegisterProcess(task.ID, cmd)
    go a.monitorProcess(cmd, task.ID, phase, config)
    return nil
}

func (a *Adapter) monitorProcess(cmd *exec.Cmd, taskID string, phase entity.Phase, config entity.PhaseConfig) {
    defer a.UnregisterProcess(taskID)
    // Monitora processo e gerencia timeouts usando cmd.Wait() ou context.Done()
    // Caso ocorra falha, dispara a lógica de retry controlada pelo orchestrator
}
```

### 4.4 Command Builder

```go
// internal/adapter/out/harness/command_builder.go
type CommandBuilder struct {
    mcpPort string
}

func (b *CommandBuilder) Build(ctx context.Context, config entity.PhaseConfig, taskID string, prompt string) (*exec.Cmd, error) {
    cmd := exec.CommandContext(ctx, config.HarnessCmd, "--model", config.ModelName, "--prompt", prompt)
    cmd.Env = append(os.Environ(), 
        fmt.Sprintf("KANBANAI_TASK_ID=%s", taskID),
        fmt.Sprintf("KANBANAI_MCP_PORT=%s", b.mcpPort),
        fmt.Sprintf("KANBANAI_MCP_URL=http://localhost:%s/mcp/sse", b.mcpPort),
    )
    return cmd, nil
}
```

---

## 5. Bootstrap e Fiação de Dependências

O arquivo `internal/adapter/bootstrap/bootstrap.go` é responsável por inicializar todas as dependências do sistema e registrar os observadores (listeners de eventos):

```go
package bootstrap

func Initialize(cfg *config.Config) (*di.Container, error) {
    container := di.NewContainer()
    
    // 1. Logger (slog)
    // 2. Conectar ao SQLite
    // 3. Registrar Repositories e Queries
    // 4. Registrar Dispatcher de Eventos e SSE Broker
    // 5. Registrar Harness Adapter e Prompt Builder
    // 6. Registrar Use Cases
    // 7. Inicializar o PhaseOrchestrator e Registrar Assinaturas do Dispatcher

    // Fiação reativa via Observer Pattern:
    
    // task.created -> StartFlow
    dispatcher.Subscribe(event.TaskCreated, func(evt event.Event) {
        task, _ := taskRepo.Find(ctx, evt.TaskID)
        orchestrator.StartFlow(ctx, task)
    })

    // task.deleted -> KillProcess
    dispatcher.Subscribe(event.TaskDeleted, func(evt event.Event) {
        orchestrator.KillProcess(evt.TaskID)
    })

    // phase.*.completed -> AdvancePhase (para cada fase)
    dispatcher.Subscribe(event.PhasePlanningCompleted, func(evt event.Event) {
        orchestrator.AdvancePhase(ctx, evt.TaskID)
    })
    // ... mesmo para Todo, Doing, Validating, Testing

    return container, nil
}
```

### 5.1 Nomes de Registro no Container

| Nome                    | Tipo                              |
|-------------------------|-----------------------------------|
| `logger`                | `*slog.Logger`                    |
| `db`                    | `*sql.DB`                         |
| `taskRepo`              | `repository.TaskRepository`       |
| `eventLogRepo`          | `repository.TaskEventLogRepository`|
| `phaseOutputRepo`       | `repository.PhaseOutputRepository`|
| `taskWithPhasesQuery`   | `query.TaskWithPhasesQuery`       |
| `taskTimelineQuery`     | `query.TaskTimelineQuery`         |
| `dispatcher`            | `event.Dispatcher`                |
| `sseBroker`             | `*sse.Broker`                     |
| `promptBuilder`         | `*service.PromptBuilder`          |
| `harnessAdapter`        | `port.HarnessPort`                |
| `createTaskUseCase`     | `*usecase.CreateTask`             |
| `updateTaskUseCase`     | `*usecase.UpdateTask`             |
| `deleteTaskUseCase`     | `*usecase.DeleteTask`             |
| `getTaskUseCase`        | `*usecase.GetTask`                |
| `listTasksUseCase`      | `*usecase.ListTasks`              |
| `advancePhaseUseCase`   | `*usecase.AdvancePhase`           |
| `reportProgressUseCase` | `*usecase.ReportPhaseProgress`    |
| `savePhaseOutputUseCase`| `*usecase.SavePhaseOutput`        |
| `orchestrator`          | `*service.PhaseOrchestrator`      |
