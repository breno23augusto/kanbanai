# KanbanAI — Domínio

## 1. Entidades do Domínio

### 1.1 Task

```go
// internal/domain/entity/task.go
type Task struct {
    ID           string
    Title        string
    Description  string
    CurrentPhase Phase
    Status       Status
    Priority     int
    Version      int       // Optimistic locking version
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### 1.2 Phase (Enum)

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

### 1.3 Status (Enum de Status da Task)

```go
// internal/domain/entity/task_status.go
type Status string

const (
    StatusPending    Status = "pending"     // Aguardando início do processamento da fase
    StatusInProgress Status = "in_progress" // Harness executando a fase ativamente
    StatusCompleted  Status = "completed"  // Fase concluída (aguardando transição ou final)
    StatusFailed     Status = "failed"     // Falha terminal após esgotar retries
    StatusCancelled  Status = "cancelled"  // Cancelado manualmente pelo usuário
)

// Transições válidas de Status:
// - StatusPending    -> StatusInProgress, StatusCancelled
// - StatusInProgress -> StatusCompleted, StatusFailed, StatusCancelled
// - StatusCompleted  -> StatusPending (ao ir para a próxima fase) ou status terminal
// - StatusFailed e StatusCancelled são terminais (só saem via intervenção manual)
```

### 1.4 TaskEventLog

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

### 1.5 PhaseConfig

```go
// internal/domain/entity/phase_config.go
type PhaseConfig struct {
    Phase       Phase
    ModelName   string    // ex: "claude-sonnet-4-20250514"
    HarnessCmd  string    // ex: "claude"
    MaxRetries  int       // Quantidade máxima de tentativas automáticas em caso de erro
    TimeoutSec  int       // Limite de execução por tentativa do harness
}
```

### 1.6 PhaseOutput (Entidade de Output da Fase)

```go
// internal/domain/entity/phase_output.go
type PhaseOutput struct {
    ID        string
    TaskID    string
    Phase     Phase
    Output    string    // Conteúdo bruto gerado (markdown, código, JSON, etc)
    Summary   string    // Resumo legível da entrega
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

---

## 2. Interfaces (Ports)

### 2.1 Criteria Pattern para Repositories

```go
// internal/domain/repository/criteria.go
type Operator string

const (
    OpEquals              Operator = "="
    OpNotEquals           Operator = "!="
    OpGreaterThan         Operator = ">"
    OpLessThan            Operator = "<"
    OpGreaterThanOrEquals Operator = ">="
    OpLessThanOrEquals    Operator = "<="
    OpLike                Operator = "LIKE"
    OpIn                  Operator = "IN"
)

type Criterion struct {
    Key      string
    Value    any
    Operator Operator
}

type Criteria []Criterion
```

### 2.2 Repositories (Porta de Saída)

```go
// internal/domain/repository/task_repository.go
type TaskRepository interface {
    Create(ctx context.Context, task *entity.Task) error
    Update(ctx context.Context, task *entity.Task) error
    Delete(ctx context.Context, id string) error
    Find(ctx context.Context, id string) (*entity.Task, error)
    FindByFilters(ctx context.Context, criteria Criteria) ([]*entity.Task, error)
}
```

```go
// internal/domain/repository/task_event_log_repository.go
type TaskEventLogRepository interface {
    Create(ctx context.Context, log *entity.TaskEventLog) error
    Update(ctx context.Context, log *entity.TaskEventLog) error
    Delete(ctx context.Context, id string) error
    Find(ctx context.Context, id string) (*entity.TaskEventLog, error)
    FindByFilters(ctx context.Context, criteria Criteria) ([]*entity.TaskEventLog, error)
}
```

```go
// internal/domain/repository/phase_output_repository.go
type PhaseOutputRepository interface {
    Create(ctx context.Context, output *entity.PhaseOutput) error
    Update(ctx context.Context, output *entity.PhaseOutput) error
    Delete(ctx context.Context, id string) error
    Find(ctx context.Context, id string) (*entity.PhaseOutput, error)
    FindByFilters(ctx context.Context, criteria Criteria) ([]*entity.PhaseOutput, error)
}
```

### 2.3 Queries Customizadas (Porta de Saída)

Queries com joins ficam em interfaces separadas:

```go
// internal/domain/query/task_with_phases_query.go
type TaskWithPhasesResult struct {
    Task         entity.Task
    PhaseOutputs []entity.PhaseOutput
}

type TaskWithPhasesQuery interface {
    Get(ctx context.Context, taskID string) (*TaskWithPhasesResult, error)
    List(ctx context.Context, criteria repository.Criteria) ([]*TaskWithPhasesResult, error)
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

### 2.4 Portas de Saída (Infraestrutura)

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

## 3. Eventos do Sistema

Os eventos cobrem todo o ciclo de vida de uma task, garantindo rastreabilidade completa.

### 3.1 Eventos de Ciclo de Vida da Task

| Evento                        | Momento de Disparo                                    |
|-------------------------------|-------------------------------------------------------|
| `task.created`                | Task registrada no banco de dados                     |
| `task.updated`                | Qualquer atualização nos dados da task                |
| `task.deleted`                | Task removida do sistema                              |
| `task.status_changed`         | Mudança de status/raia da task                        |

### 3.2 Eventos de Transição de Raia

| Evento                        | Momento de Disparo                                    |
|-------------------------------|-------------------------------------------------------|
| `lane.transition.started`     | Início da transição entre raias                       |
| `lane.transition.completed`   | Transição concluída com sucesso                       |
| `lane.transition.failed`      | Falha na transição                                    |

### 3.3 Eventos de Fase (por raia)

Todos os eventos de fase seguem uma estrutura previsível e são disparados em todas as fases não terminais (planning, todo, doing, validating, testing).

| Evento                        | Momento de Disparo                                    |
|-------------------------------|-------------------------------------------------------|
| `phase.planning.started`      | Harness iniciou a fase de planning                    |
| `phase.planning.progress`     | Harness reportou progresso parcial na fase            |
| `phase.planning.retry`        | Tentativa falhou, executando retry da fase            |
| `phase.planning.completed`    | Harness finalizou planning com sucesso                |
| `phase.planning.failed`       | Falha terminal na fase de planning                    |
| `phase.todo.started`          | Início da fase Todo                                   |
| `phase.todo.progress`         | Progresso reportado na fase Todo                      |
| `phase.todo.retry`            | Tentativa falhou, executando retry de Todo            |
| `phase.todo.completed`        | Fase Todo concluída                                   |
| `phase.todo.failed`           | Falha terminal na fase Todo                           |
| `phase.doing.started`         | Início da fase Doing                                  |
| `phase.doing.progress`        | Progresso reportado na fase Doing                     |
| `phase.doing.retry`           | Tentativa falhou, executando retry de Doing           |
| `phase.doing.completed`       | Fase Doing concluída                                  |
| `phase.doing.failed`          | Falha terminal na fase Doing                          |
| `phase.validating.started`    | Início da fase Validating                             |
| `phase.validating.progress`   | Progresso reportado na fase Validating                |
| `phase.validating.retry`      | Tentativa falhou, executando retry de Validating      |
| `phase.validating.completed`  | Fase Validating concluída                             |
| `phase.validating.failed`     | Falha terminal na fase Validating                     |
| `phase.testing.started`       | Início da fase Testing                                |
| `phase.testing.progress`      | Progresso reportado na fase Testing                   |
| `phase.testing.retry`         | Tentativa falhou, executando retry de Testing         |
| `phase.testing.completed`     | Fase Testing concluída                                |
| `phase.testing.failed`        | Falha terminal na fase Testing                        |
| `phase.done.reached`          | Task chegou na raia final Done                        |

### 3.4 Eventos de Harness

| Evento                        | Momento de Disparo                                    |
|-------------------------------|-------------------------------------------------------|
| `harness.command.dispatched`  | Comando enviado ao harness                            |
| `harness.command.acknowledged`| Harness confirmou recebimento                         |
| `harness.output.received`     | Saída parcial recebida do harness                     |
| `harness.error.occurred`      | Erro no harness                                       |
| `harness.session.started`     | Sessão MCP iniciada                                   |
| `harness.session.ended`       | Sessão MCP encerrada                                  |

### 3.5 Eventos de Sistema

| Evento                        | Momento de Disparo                                    |
|-------------------------------|-------------------------------------------------------|
| `system.health.check`         | Health check periódico                                |
| `system.error`                | Erro genérico do sistema                              |
| `sse.client.connected`        | Cliente SSE conectou                                  |
| `sse.client.disconnected`     | Cliente SSE desconectou                               |
