# KanbanAI — Especificação Técnica

## 1. Visão Geral

**KanbanAI** é uma aplicação em Go que orquestra um fluxo Kanban automatizado por inteligência artificial. O sistema conecta-se a harnesses de IA (Claude, Pi, Hermes etc.) via **MCP (Model Context Protocol)** para executar cada fase do fluxo de forma autônoma.

O usuário cria uma task pelo dashboard; o sistema persiste no banco SQLite, dispara eventos via **Observer Pattern**, notifica o frontend em tempo real via **SSE (Server-Sent Events)** e coordena a execução de cada raia do Kanban através de um harness de IA configurável.

---

## 2. Raias do Kanban

O fluxo segue a ordem estrita abaixo. Cada raia representa uma fase da execução:

| Ordem | Raia           | Descrição                                                        |
|-------|----------------|------------------------------------------------------------------|
| 1     | **Planning**    | Planejamento da task: escopo, subtasks, critérios de aceite      |
| 2     | **Todo**        | Backlog refinado e pronto para execução                          |
| 3     | **Doing**       | Implementação ativa pela harness                                 |
| 4     | **Validating**  | Revisão de código e validação de requisitos                      |
| 5     | **Testing**     | Criação e execução de testes automatizados e manuais                       |
| 6     | **Done**        | Task concluída e entregue                                        |

Cada raia pode ser configurada com um **modelo de IA diferente** via variáveis de ambiente.

---

## 3. Stack Tecnológica

| Camada           | Tecnologia                              | Versão / Detalhes          |
|------------------|-----------------------------------------|----------------------------|
| Linguagem        | Go                                      | 1.26                       |
| HTTP Framework   | gin-gonic/gin                           | latest                     |
| MCP SDK          | modelcontextprotocol/go-sdk             | latest                     |
| CLI              | spf13/cobra                             | latest                     |
| Configuração     | spf13/viper                             | latest                     |
| Banco de Dados   | SQLite (via mattn/go-sqlite3)           | latest                     |
| Testes / Mocks   | stretchr/testify                        | latest                     |
| Frontend         | React + Material UI                     | React 18+, MUI 5+          |
| Comunicação RT   | SSE (Server-Sent Events)                | nativo                     |

---

## 4. Princípios e Padrões

### 4.1 Princípios SOLID

- **S** — Single Responsibility: cada arquivo e struct tem um único propósito.
- **O** — Open/Closed: extensão via interfaces, não modificação de structs concretas.
- **L** — Liskov Substitution: qualquer implementação de interface pode substituir outra sem quebra.
- **I** — Interface Segregation: interfaces pequenas e focadas (ex: `TaskCreator`, `TaskFinder`).
- **D** — Dependency Inversion: domínio define interfaces, infraestrutura implementa.

### 4.2 Clean Code

- Nomes descritivos e consistentes em inglês.
- Funções pequenas (máximo ~30 linhas).
- Sem comentários óbvios — o código deve ser autoexplicativo.
- Tratamento de erros explícito e contextualizado com `fmt.Errorf("context: %w", err)`.

### 4.3 Arquitetura Hexagonal (Ports & Adapters)

```
┌──────────────────────────────────────────────────────────┐
│                     ADAPTERS (IN)                        │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────┐ │
│  │ HTTP (Gin) │  │  CLI (Cobra)│  │  MCP Server (SDK) │ │
│  └─────┬──────┘  └─────┬──────┘  └────────┬───────────┘ │
│        │               │                   │             │
│  ══════╪═══════════════╪═══════════════════╪══════════   │
│        │         PORTS (IN)                │             │
│        │    ┌──────────────────┐           │             │
│        └───►│   Use Cases      │◄──────────┘             │
│             │  (Application)   │                         │
│             └────────┬─────────┘                         │
│                      │                                   │
│  ════════════════════╪═══════════════════════════════    │
│               PORTS (OUT)                                │
│             ┌────────┴─────────┐                         │
│             │    Domain        │                         │
│             │  (Entities +     │                         │
│             │   Interfaces)    │                         │
│             └────────┬─────────┘                         │
│                      │                                   │
│  ════════════════════╪═══════════════════════════════    │
│              ADAPTERS (OUT)                              │
│  ┌────────────┐  ┌────────────┐  ┌──────────────────┐   │
│  │  SQLite    │  │  Harness   │  │  Event Emitter   │   │
│  │  Repos     │  │  Client    │  │  (Observer)      │   │
│  └────────────┘  └────────────┘  └──────────────────┘   │
└──────────────────────────────────────────────────────────┘
```

### 4.4 Injeção de Dependências (DI In-Memory)

Um container simples em memória gerencia todas as dependências:

```go
// internal/di/container.go
type Container struct {
    mu       sync.RWMutex
    services map[string]any
}

func (c *Container) Register(name string, svc any)
func (c *Container) Resolve(name string) any
func (c *Container) MustResolve(name string) any
```

Todas as dependências são registradas no bootstrap da aplicação e resolvidas via container. Nenhum `new` direto em use cases.

### 4.5 Observer Pattern

O sistema de eventos é o coração da reatividade:

```go
// internal/domain/event/dispatcher.go
type EventType string

type Event struct {
    Type      EventType
    Payload   any
    Timestamp time.Time
    TaskID    string
}

type Handler func(event Event)

type Dispatcher interface {
    Subscribe(eventType EventType, handler Handler)
    Publish(event Event)
}
```

---

## 5. Eventos do Sistema

Os eventos cobrem todo o ciclo de vida de uma task, garantindo rastreabilidade completa:

### 5.1 Eventos de Ciclo de Vida da Task

| Evento                        | Momento de Disparo                                    |
|-------------------------------|-------------------------------------------------------|
| `task.created`                | Task registrada no banco de dados                     |
| `task.updated`                | Qualquer atualização nos dados da task                |
| `task.deleted`                | Task removida do sistema                              |
| `task.status_changed`         | Mudança de status/raia da task                        |

### 5.2 Eventos de Transição de Raia

| Evento                        | Momento de Disparo                                    |
|-------------------------------|-------------------------------------------------------|
| `lane.transition.started`     | Início da transição entre raias                       |
| `lane.transition.completed`   | Transição concluída com sucesso                       |
| `lane.transition.failed`      | Falha na transição                                    |

### 5.3 Eventos de Fase (por raia)

| Evento                        | Momento de Disparo                                    |
|-------------------------------|-------------------------------------------------------|
| `phase.planning.started`      | Harness iniciou a fase de planning                    |
| `phase.planning.progress`     | Harness reportou progresso na fase                    |
| `phase.planning.completed`    | Harness finalizou planning com sucesso                |
| `phase.planning.failed`       | Falha na fase de planning                             |
| `phase.todo.started`          | Idem para cada raia subsequente                       |
| `phase.todo.completed`        | ...                                                   |
| `phase.doing.started`         | ...                                                   |
| `phase.doing.progress`        | ...                                                   |
| `phase.doing.completed`       | ...                                                   |
| `phase.validating.started`    | ...                                                   |
| `phase.validating.completed`  | ...                                                   |
| `phase.testing.started`       | ...                                                   |
| `phase.testing.progress`      | ...                                                   |
| `phase.testing.completed`     | ...                                                   |
| `phase.done.reached`          | Task chegou na raia Done                              |

### 5.4 Eventos de Harness

| Evento                        | Momento de Disparo                                    |
|-------------------------------|-------------------------------------------------------|
| `harness.command.dispatched`  | Comando enviado ao harness                            |
| `harness.command.acknowledged`| Harness confirmou recebimento                         |
| `harness.output.received`     | Saída parcial recebida do harness                     |
| `harness.error.occurred`      | Erro no harness                                       |
| `harness.session.started`     | Sessão MCP iniciada                                   |
| `harness.session.ended`       | Sessão MCP encerrada                                  |

### 5.5 Eventos de Sistema

| Evento                        | Momento de Disparo                                    |
|-------------------------------|-------------------------------------------------------|
| `system.health.check`         | Health check periódico                                |
| `system.error`                | Erro genérico do sistema                              |
| `sse.client.connected`        | Cliente SSE conectou                                  |
| `sse.client.disconnected`     | Cliente SSE desconectou                               |

---

## 6. Fluxo Principal

```
┌──────────┐      POST /api/tasks       ┌──────────────┐
│ Frontend │ ───────────────────────────►│ HTTP Handler │
│ (React)  │                             └──────┬───────┘
│          │◄─── SSE /api/events ────────────┐  │
└──────────┘                                 │  │
                                             │  ▼
                                    ┌────────┴──────────┐
                                    │  CreateTaskUseCase │
                                    └────────┬──────────┘
                                             │
                              ┌──────────────┼──────────────┐
                              ▼              ▼              ▼
                    ┌──────────────┐ ┌──────────────┐ ┌───────────┐
                    │ TaskRepo     │ │ EventDispatch│ │ SSE Hub   │
                    │ (SQLite)     │ │ (Observer)   │ │ (Broker)  │
                    └──────────────┘ └──────┬───────┘ └───────────┘
                                            │
                                            ▼
                                   ┌─────────────────┐
                                   │ PhaseOrchestrator│
                                   └────────┬────────┘
                                            │
                                            ▼
                                   ┌─────────────────┐
                                   │ HarnessAdapter   │
                                   │ (CLI dispatch)   │
                                   └────────┬────────┘
                                            │
                              ┌─────────────┼─────────────┐
                              ▼             ▼             ▼
                         ┌────────┐   ┌──────────┐  ┌──────────┐
                         │ Claude │   │   Pi     │  │ Hermes   │
                         └────────┘   └──────────┘  └──────────┘
```

### 6.1 Passo a Passo

1. **Usuário cria task** no dashboard (React).
2. **POST `/api/tasks`** é enviado ao servidor Go.
3. **`CreateTaskUseCase`** valida e persiste no SQLite via `TaskRepository`.
4. **Evento `task.created`** é disparado pelo `EventDispatcher`.
5. O **SSE Hub** recebe o evento e envia para todos os clientes conectados.
6. O **`PhaseOrchestrator`** (subscriber de `task.created`) inicia o fluxo.
7. Evento **`phase.planning.started`** é disparado.
8. O **`HarnessAdapter`** monta o prompt com instruções MCP e dispara o harness.
9. O **harness executa** a fase, usando o MCP para reportar progresso ao servidor.
10. Via MCP, o harness chama tools como `update_task_phase`, `report_progress`, `complete_phase`.
11. Ao chamar `complete_phase`, evento **`phase.planning.completed`** é disparado.
12. O **`PhaseOrchestrator`** reage e inicia a próxima raia (Todo).
13. **Repete** até chegar em Done.
14. Evento **`phase.done.reached`** é disparado.

### 6.2 Prompt para o Harness (exemplo para Planning)

```
Você é um agente de planejamento de software. Sua tarefa é planejar a implementação da seguinte task:

**Task**: {{ .Task.Title }}
**Descrição**: {{ .Task.Description }}

Você tem acesso a um servidor MCP para reportar seu progresso e resultados.

## Ferramentas MCP Disponíveis

- `report_progress(task_id, message)` — Reporte progresso parcial
- `update_task_output(task_id, phase, output)` — Salve artefatos da fase
- `complete_phase(task_id, phase, summary)` — Marque a fase como concluída

## Instruções

1. Analise os requisitos da task
2. Defina subtasks necessárias
3. Estabeleça critérios de aceite
4. Use `report_progress` para informar o andamento
5. Use `update_task_output` para salvar o plano
6. Use `complete_phase` para finalizar

MCP Server: {{ .MCPServerURL }}
Task ID: {{ .Task.ID }}
Phase: planning
```

---

## 7. Estrutura de Diretórios

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
│   │   │   └── phase_config.go              # Config de modelo por raia
│   │   │
│   │   ├── event/
│   │   │   ├── types.go                     # Definição dos EventTypes
│   │   │   ├── event.go                     # Struct Event
│   │   │   ├── dispatcher.go                # Interface Dispatcher
│   │   │   └── handler.go                   # Type Handler
│   │   │
│   │   ├── repository/
│   │   │   ├── task_repository.go           # Interface TaskRepository
│   │   │   └── task_event_log_repository.go # Interface TaskEventLogRepository
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
│           │   │   ├── task_repository_sqlite.go       # Impl TaskRepository
│           │   │   ├── task_repository_sqlite_test.go
│           │   │   ├── task_event_log_repository_sqlite.go
│           │   │   └── task_event_log_repository_sqlite_test.go
│           │   │
│           │   └── query/
│           │       ├── task_with_phases_query_sqlite.go  # Impl join query
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
└── SPEC.md                                  # Este documento
```

---

## 8. Entidades do Domínio

### 8.1 Task

```go
// internal/domain/entity/task.go
type Task struct {
    ID           string
    Title        string
    Description  string
    CurrentPhase Phase
    Status       Status
    Priority     int
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### 8.2 Phase (Enum)

```go
// internal/domain/entity/task_phase.go
type Phase string

const (
    PhasePlanning   Phase = "planning"
    PhaseTodo       Phase = "todo"
    PhaseDoing      Phase = "doing"
    PhaseValidating Phase = "validating"
    PhaseTesting    Phase = "testing"
    PhaseDone       Phase = "done"
)

// PhaseOrder define a ordem de execução
var PhaseOrder = []Phase{
    PhasePlanning,
    PhaseTodo,
    PhaseDoing,
    PhaseValidating,
    PhaseTesting,
    PhaseDone,
}

func (p Phase) Next() (Phase, bool)
func (p Phase) IsTerminal() bool
```

### 8.3 TaskEventLog

```go
// internal/domain/entity/task_event_log.go
type TaskEventLog struct {
    ID        string
    TaskID    string
    EventType string
    Phase     Phase
    Message   string
    Metadata  map[string]any
    CreatedAt time.Time
}
```

### 8.4 PhaseConfig

```go
// internal/domain/entity/phase_config.go
type PhaseConfig struct {
    Phase       Phase
    ModelName   string    // ex: "claude-sonnet-4-20250514"
    HarnessCmd  string    // ex: "claude"
    MaxRetries  int
    TimeoutSec  int
}
```

---

## 9. Interfaces (Ports)

### 9.1 Repositories (Porta de Saída)

Todos os repositories seguem o contrato padrão:

```go
// internal/domain/repository/task_repository.go
type TaskRepository interface {
    Create(ctx context.Context, task *entity.Task) error
    Update(ctx context.Context, task *entity.Task) error
    Delete(ctx context.Context, id string) error
    Find(ctx context.Context, id string) (*entity.Task, error)
    FindByFilters(ctx context.Context, filters TaskFilters) ([]*entity.Task, error)
}

type TaskFilters struct {
    Phase    *entity.Phase
    Status   *entity.Status
    Priority *int
    Limit    int
    Offset   int
}
```

### 9.2 Queries Customizadas (Porta de Saída)

Queries com joins ficam em interfaces separadas:

```go
// internal/domain/query/task_with_phases_query.go
type TaskWithPhasesResult struct {
    Task         entity.Task
    PhaseOutputs []PhaseOutput
}

type TaskWithPhasesQuery interface {
    Get(ctx context.Context, taskID string) (*TaskWithPhasesResult, error)
    List(ctx context.Context, filters TaskFilters) ([]*TaskWithPhasesResult, error)
}
```

```go
// internal/domain/query/task_timeline_query.go
type TaskTimelineResult struct {
    Task   entity.Task
    Events []entity.TaskEventLog
}

type TaskTimelineQuery interface {
    Get(ctx context.Context, taskID string) (*TaskTimelineResult, error)
}
```

### 9.3 Portas de Saída (Infraestrutura)

```go
// internal/domain/port/harness_port.go
type HarnessPort interface {
    Dispatch(ctx context.Context, task *entity.Task, phase entity.Phase, prompt string) error
}
```

```go
// internal/domain/port/sse_port.go
type SSEPort interface {
    Broadcast(event event.Event)
    Subscribe(clientID string) <-chan event.Event
    Unsubscribe(clientID string)
}
```

```go
// internal/domain/port/phase_orchestrator_port.go
type PhaseOrchestratorPort interface {
    StartFlow(ctx context.Context, task *entity.Task) error
    AdvancePhase(ctx context.Context, taskID string) error
}
```

---

## 10. Use Cases (Application Layer)

Cada use case é um arquivo dedicado com uma struct e um método `Execute`:

```go
// internal/application/usecase/create_task.go
type CreateTask struct {
    taskRepo   repository.TaskRepository
    dispatcher event.Dispatcher
}

func NewCreateTask(repo repository.TaskRepository, disp event.Dispatcher) *CreateTask

func (uc *CreateTask) Execute(ctx context.Context, input dto.CreateTaskInput) (*dto.TaskOutput, error) {
    // 1. Validar input
    // 2. Criar entidade Task
    // 3. Persistir via taskRepo.Create()
    // 4. Disparar evento task.created
    // 5. Retornar DTO de saída
}
```

### 10.1 Lista de Use Cases

| Use Case                | Arquivo                      | Responsabilidade                           |
|-------------------------|------------------------------|--------------------------------------------|
| `CreateTask`            | `create_task.go`             | Criar task e disparar fluxo                |
| `UpdateTask`            | `update_task.go`             | Atualizar dados da task                    |
| `DeleteTask`            | `delete_task.go`             | Remover task                               |
| `GetTask`               | `get_task.go`                | Buscar task por ID                         |
| `ListTasks`             | `list_tasks.go`              | Listar tasks com filtros                   |
| `AdvancePhase`          | `advance_phase.go`           | Transicionar task para próxima fase        |
| `ReportPhaseProgress`   | `report_phase_progress.go`   | Registrar progresso de uma fase            |

---

## 11. MCP Server — Tools

O servidor MCP expõe tools para que o harness interaja com o sistema:

### 11.1 Tools Disponíveis

| Tool                    | Método     | Descrição                                            |
|-------------------------|------------|------------------------------------------------------|
| `get_task`              | Read       | Busca informações da task atual                      |
| `report_progress`       | Write      | Reporta progresso parcial da fase em execução        |
| `update_task_output`    | Write      | Salva artefatos/outputs da fase (plano, código etc.) |
| `complete_phase`        | Write      | Marca fase como concluída, dispara próxima           |

### 11.2 Implementação

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
        Description: "Marks the current phase as completed and triggers the next phase",
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
            advancePhase := container.MustResolve("advancePhase").(*usecase.AdvancePhase)
            return advancePhase.Execute(ctx, args["task_id"].(string))
        },
    }
}
```

---

## 12. SSE (Server-Sent Events)

### 12.1 Broker

```go
// internal/adapter/out/sse/broker.go
type Broker struct {
    mu          sync.RWMutex
    clients     map[string]chan event.Event
    dispatcher  event.Dispatcher
}

func NewBroker(dispatcher event.Dispatcher) *Broker

func (b *Broker) Broadcast(evt event.Event)
func (b *Broker) Subscribe(clientID string) <-chan event.Event
func (b *Broker) Unsubscribe(clientID string)
```

### 12.2 Handler HTTP

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

## 13. Harness Adapter

### 13.1 Dispatch de Comando

```go
// internal/adapter/out/harness/adapter.go
type Adapter struct {
    configs    map[entity.Phase]entity.PhaseConfig
    builder    *CommandBuilder
    dispatcher event.Dispatcher
}

func (a *Adapter) Dispatch(ctx context.Context, task *entity.Task, phase entity.Phase, prompt string) error {
    config := a.configs[phase]

    a.dispatcher.Publish(event.Event{
        Type:    event.HarnessCommandDispatched,
        TaskID:  task.ID,
        Payload: map[string]any{"phase": phase, "model": config.ModelName},
    })

    cmd := a.builder.Build(config, prompt)
    return cmd.Start() // executa em background
}
```

### 13.2 Command Builder

```go
// internal/adapter/out/harness/command_builder.go
type CommandBuilder struct{}

func (b *CommandBuilder) Build(config entity.PhaseConfig, prompt string) *exec.Cmd {
    // Ex: claude --model claude-sonnet-4-20250514 --prompt "..."
    return exec.Command(config.HarnessCmd, "--model", config.ModelName, "--prompt", prompt)
}
```

---

## 14. Configuração de Ambiente

### 14.1 Variáveis de Ambiente

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

# Frontend
KANBANAI_FRONTEND_URL=http://localhost:3000

# Log Level
KANBANAI_LOG_LEVEL=info
```

### 14.2 Struct de Configuração

```go
// config/config.go
type Config struct {
    Server  ServerConfig
    DB      DBConfig
    MCP     MCPConfig
    Harness HarnessConfig
    Log     LogConfig
}

type HarnessConfig struct {
    DefaultCmd   string
    DefaultModel string
    Phases       map[entity.Phase]PhaseHarnessConfig
}

type PhaseHarnessConfig struct {
    Cmd   string
    Model string
}
```

---

## 15. CLI (Cobra)

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

---

## 16. API REST

### 16.1 Endpoints

| Método   | Rota                        | Handler                | Descrição                    |
|----------|-----------------------------|------------------------|------------------------------|
| `GET`    | `/api/health`               | `HealthHandler`        | Health check                 |
| `POST`   | `/api/tasks`                | `TaskHandler.Create`   | Criar nova task              |
| `GET`    | `/api/tasks`                | `TaskHandler.List`     | Listar tasks (com filtros)   |
| `GET`    | `/api/tasks/:id`            | `TaskHandler.Get`      | Buscar task por ID           |
| `PUT`    | `/api/tasks/:id`            | `TaskHandler.Update`   | Atualizar task               |
| `DELETE` | `/api/tasks/:id`            | `TaskHandler.Delete`   | Deletar task                 |
| `GET`    | `/api/tasks/:id/timeline`   | `TaskHandler.Timeline` | Timeline de eventos da task  |
| `GET`    | `/api/events`               | `SSEHandler.Stream`    | Stream SSE de eventos        |

### 16.2 Formato de Response

```json
// Sucesso
{
    "success": true,
    "data": { "..." : "..." },
    "meta": {
        "request_id": "01J...",
        "timestamp": "2026-06-23T20:00:00Z"
    }
}

// Erro
{
    "success": false,
    "error": {
        "code": "TASK_NOT_FOUND",
        "message": "Task with ID '...' not found"
    },
    "meta": {
        "request_id": "01J...",
        "timestamp": "2026-06-23T20:00:00Z"
    }
}
```

---

## 17. Frontend (React + MUI)

### 17.1 Componentes Principais

| Componente           | Responsabilidade                                           |
|----------------------|------------------------------------------------------------|
| `KanbanBoard`        | Grid com as 6 raias, drag indicator visual                 |
| `KanbanLane`         | Coluna individual com header, contagem e cards             |
| `TaskCard`           | Card com título, status, fase atual e indicador de progresso |
| `CreateTaskDialog`   | Dialog MUI para criar nova task                            |
| `TaskDetailDrawer`   | Drawer lateral com detalhes, outputs e timeline            |
| `EventTimeline`      | Timeline vertical dos eventos da task                      |
| `PhaseProgress`      | Stepper MUI mostrando progresso entre fases                |

### 17.2 Hook SSE

```typescript
// frontend/src/hooks/useSSE.ts
export function useSSE(url: string) {
    const [events, setEvents] = useState<SSEEvent[]>([]);

    useEffect(() => {
        const source = new EventSource(url);

        source.addEventListener('task.created', (e) => { /* ... */ });
        source.addEventListener('phase.planning.started', (e) => { /* ... */ });
        // ... demais eventos

        return () => source.close();
    }, [url]);

    return { events };
}
```

---

## 18. Testes

### 18.1 Estratégia

- **Unit Tests**: todos os use cases, services e adapters com mocks via `testify/mock`.
- **Integration Tests**: repositories contra SQLite in-memory.
- **E2E**: fluxo completo criação → done com harness mockado.

### 18.2 Mocks com Testify

```go
// internal/domain/repository/mock/task_repository_mock.go
type MockTaskRepository struct {
    mock.Mock
}

func (m *MockTaskRepository) Create(ctx context.Context, task *entity.Task) error {
    args := m.Called(ctx, task)
    return args.Error(0)
}

func (m *MockTaskRepository) Find(ctx context.Context, id string) (*entity.Task, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(*entity.Task), args.Error(1)
}

// ... demais métodos
```

### 18.3 Exemplo de Teste de Use Case

```go
// internal/application/usecase/create_task_test.go
func TestCreateTask_Execute_Success(t *testing.T) {
    mockRepo := new(mocks.MockTaskRepository)
    mockDispatcher := new(mocks.MockDispatcher)

    mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Task")).Return(nil)
    mockDispatcher.On("Publish", mock.AnythingOfType("event.Event")).Return()

    uc := usecase.NewCreateTask(mockRepo, mockDispatcher)

    input := dto.CreateTaskInput{
        Title:       "Implementar login",
        Description: "OAuth2 com Google",
    }

    result, err := uc.Execute(context.Background(), input)

    assert.NoError(t, err)
    assert.NotEmpty(t, result.ID)
    assert.Equal(t, entity.PhasePlanning, result.CurrentPhase)
    mockRepo.AssertExpectations(t)
    mockDispatcher.AssertExpectations(t)
}
```

---

## 19. Schema do Banco (SQLite)

```sql
-- migration_files/001_create_tasks.sql
CREATE TABLE IF NOT EXISTS tasks (
    id            TEXT PRIMARY KEY,
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    current_phase TEXT NOT NULL DEFAULT 'planning',
    status        TEXT NOT NULL DEFAULT 'pending',
    priority      INTEGER NOT NULL DEFAULT 0,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tasks_current_phase ON tasks(current_phase);
CREATE INDEX idx_tasks_status ON tasks(status);
```

```sql
-- migration_files/002_create_task_event_logs.sql
CREATE TABLE IF NOT EXISTS task_event_logs (
    id         TEXT PRIMARY KEY,
    task_id    TEXT NOT NULL,
    event_type TEXT NOT NULL,
    phase      TEXT,
    message    TEXT NOT NULL DEFAULT '',
    metadata   TEXT DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX idx_task_event_logs_task_id ON task_event_logs(task_id);
CREATE INDEX idx_task_event_logs_event_type ON task_event_logs(event_type);
```

```sql
-- migration_files/003_create_phase_outputs.sql
CREATE TABLE IF NOT EXISTS phase_outputs (
    id         TEXT PRIMARY KEY,
    task_id    TEXT NOT NULL,
    phase      TEXT NOT NULL,
    output     TEXT NOT NULL DEFAULT '',
    summary    TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    UNIQUE(task_id, phase)
);

CREATE INDEX idx_phase_outputs_task_id ON phase_outputs(task_id);
```

---

## 20. Makefile

```makefile
.PHONY: build run test lint migrate dev

build:
	go build -o bin/kanbanai cmd/kanbanai/main.go

run: build
	./bin/kanbanai serve

dev:
	go run cmd/kanbanai/main.go serve

test:
	go test ./... -v -race -cover

lint:
	golangci-lint run ./...

migrate:
	go run cmd/kanbanai/main.go migrate

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build
```

---

## 21. Convenções de Nomenclatura

| Elemento             | Convenção                          | Exemplo                               |
|----------------------|------------------------------------|---------------------------------------|
| Package              | lowercase singular                 | `usecase`, `handler`, `entity`        |
| Interface            | PascalCase, noun                   | `TaskRepository`, `Dispatcher`        |
| Struct               | PascalCase, noun                   | `CreateTask`, `SSEBroker`             |
| Método               | PascalCase, verb+noun              | `Execute`, `FindByFilters`            |
| Arquivo              | snake_case                         | `create_task.go`, `task_handler.go`   |
| Teste                | `<filename>_test.go`               | `create_task_test.go`                 |
| Constante            | PascalCase com prefixo do tipo     | `PhasePlanning`, `StatusActive`       |
| Variável de ambiente | UPPER_SNAKE_CASE com prefixo       | `KANBANAI_SERVER_PORT`                |
| Evento               | dot.notation lowercase             | `task.created`, `phase.planning.started` |
| DTO                  | PascalCase + Input/Output/Filter   | `CreateTaskInput`, `TaskOutput`       |
| Repository impl      | Interface + Storage suffix         | `TaskRepositorySQLite`                |
| Query impl           | Interface + Storage suffix         | `TaskWithPhasesQuerySQLite`           |
| Mock                 | Mock + Interface name              | `MockTaskRepository`                  |

---

## 22. Performance — Boas Práticas Go

- **Goroutines**: O `PhaseOrchestrator` executa harness dispatch em goroutines separadas.
- **Channels**: SSE Broker usa channels para comunicação non-blocking entre goroutines.
- **sync.Pool**: Reutilização de buffers no formatter SSE.
- **Context propagation**: Todos os métodos recebem `context.Context` para cancelamento e timeouts.
- **Prepared statements**: SQLite queries usam prepared statements cacheados.
- **Batch inserts**: Event logs podem ser inseridos em batch quando há muitos eventos.
- **Read-Write Mutex**: `sync.RWMutex` em caches e maps compartilhados.
- **Struct embedding**: Uso de composição ao invés de herança para reutilização.

---

## 23. Diagrama de Sequência — Fluxo Completo

```
User        Frontend       HTTP         UseCase      Repository    Dispatcher    SSE         Orchestrator    Harness
 │              │            │              │             │             │          │              │              │
 │──create──►   │            │              │             │             │          │              │              │
 │              │──POST──►   │              │             │             │          │              │              │
 │              │            │──Execute──►  │             │             │          │              │              │
 │              │            │              │──Create──►  │             │          │              │              │
 │              │            │              │             │──persist──► │          │              │              │
 │              │            │              │             │◄──ok────── │          │              │              │
 │              │            │              │──Publish──► │             │          │              │              │
 │              │            │              │             │  task.created          │              │              │
 │              │            │              │             │             │──SSE──►  │              │              │
 │              │◄──SSE─────────────────────────────────────────────── │          │              │              │
 │              │            │              │             │             │──notify──►              │              │
 │              │            │              │             │             │          │──StartFlow──►│              │
 │              │            │              │             │             │          │              │──Dispatch──► │
 │              │            │              │             │             │          │              │              │──run harness
 │              │            │              │             │             │          │              │              │
 │              │            │          ◄────────────────MCP: report_progress─────────────────── │              │
 │              │            │              │             │             │──SSE──►  │              │              │
 │              │◄──SSE─────────────────────────────────────────────── │          │              │              │
 │              │            │          ◄────────────────MCP: complete_phase──────────────────── │              │
 │              │            │              │──Publish──► │             │          │              │              │
 │              │            │              │    phase.planning.completed          │              │              │
 │              │            │              │             │             │──notify──►              │              │
 │              │            │              │             │             │          │──next phase──►              │
 │              │            │              │             │             │          │              │   (repeat)   │
```

---

## 24. Resumo de Decisões Arquiteturais

| Decisão                               | Justificativa                                                       |
|---------------------------------------|---------------------------------------------------------------------|
| Hexagonal Architecture                | Desacoplamento total entre domínio e infraestrutura                 |
| Observer Pattern                      | Reatividade sem acoplamento direto entre componentes                |
| SSE ao invés de WebSocket             | Mais simples, unidirecional (server→client), suficiente para o caso |
| SQLite                                | Lightweight, zero-config, suficiente para o escopo                  |
| Repository + Query separados          | SRP: CRUD simples vs queries complexas com joins                    |
| DI Container in-memory                | Simplicidade sem framework pesado                                   |
| Arquivo por use case                  | SRP + facilita navegação e testes                                   |
| Modelo por raia configurável          | Flexibilidade para usar modelos especializados por fase             |
| MCP para comunicação harness→server   | Protocolo padrão para comunicação com LLMs                          |
| Cobra + Viper                         | Stack padrão Go para CLI + config                                   |
