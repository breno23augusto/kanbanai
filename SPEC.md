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
    SubscribeAll(handler Handler)  // Wildcard: recebe todos os eventos publicados
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

### 6.3 Lógica Interna do PhaseOrchestrator

O `PhaseOrchestrator` é o serviço central que coordena a execução das fases. Ele **não** é um use case — é um serviço de aplicação que reage a eventos e gerencia o ciclo de vida dos processos harness.

```go
// internal/application/service/phase_orchestrator.go
type PhaseOrchestrator struct {
    taskRepo        repository.TaskRepository
    phaseOutputRepo repository.PhaseOutputRepository
    harnessAdapter  port.HarnessPort
    promptBuilder   *PromptBuilder
    dispatcher      event.Dispatcher
    processRegistry map[string]*exec.Cmd  // taskID → harness process
    mu              sync.RWMutex
}
```

#### 6.3.1 StartFlow — Início do Fluxo

Iniciado quando o evento `task.created` é disparado. Começa sempre pela fase `Planning`:

```go
func (o *PhaseOrchestrator) StartFlow(ctx context.Context, task *entity.Task) error {
    o.dispatcher.Publish(event.Event{
        Type: event.PhasePlanningStarted, TaskID: task.ID,
        Payload: map[string]any{"phase": entity.PhasePlanning},
    })
    prompt, err := o.promptBuilder.Build(entity.PhasePlanning, task)
    if err != nil {
        return fmt.Errorf("prompt build: %w", err)
    }
    if err := o.harnessAdapter.Dispatch(ctx, task, entity.PhasePlanning, prompt); err != nil {
        return fmt.Errorf("harness dispatch: %w", err)
    }
    return nil
}
```

#### 6.3.2 AdvancePhase — Avançar para Próxima Fase

Chamado pelos subscribers de eventos `phase.*.completed`. **Não** é o mesmo que o use case `AdvancePhase` (veja distinção em 6.3.5):

```go
func (o *PhaseOrchestrator) AdvancePhase(ctx context.Context, taskID string) error {
    task, err := o.taskRepo.Find(ctx, taskID)
    if err != nil {
        return fmt.Errorf("find task: %w", err)
    }
    nextPhase, hasNext := task.CurrentPhase.Next()
    if !hasNext || task.CurrentPhase.IsTerminal() {
        task.Status = entity.StatusCompleted
        task.UpdatedAt = time.Now()
        if err := o.taskRepo.Update(ctx, task); err != nil {
            return fmt.Errorf("update task: %w", err)
        }
        o.dispatcher.Publish(event.Event{
            Type: event.PhaseDoneReached, TaskID: task.ID,
        })
        return nil
    }
    task.CurrentPhase = nextPhase
    task.Status = entity.StatusPending
    task.UpdatedAt = time.Now()
    if err := o.taskRepo.Update(ctx, task); err != nil {
        return fmt.Errorf("update task: %w", err)
    }
    o.dispatcher.Publish(event.Event{
        Type: event.LaneTransitionCompleted, TaskID: task.ID,
        Payload: map[string]any{"from": task.CurrentPhase, "to": nextPhase},
    })
    return o.dispatchPhase(ctx, task, nextPhase)
}
```

#### 6.3.3 dispatchPhase (método interno)

```go
func (o *PhaseOrchestrator) dispatchPhase(ctx context.Context, task *entity.Task, phase entity.Phase) error {
    task.Status = entity.StatusInProgress
    task.UpdatedAt = time.Now()
    if err := o.taskRepo.Update(ctx, task); err != nil {
        return fmt.Errorf("update task status: %w", err)
    }
    o.dispatcher.Publish(event.Event{
        Type: event.PhaseEvent(phase, "started"), TaskID: task.ID,
        Payload: map[string]any{"phase": phase},
    })
    prompt, err := o.promptBuilder.Build(phase, task)
    if err != nil {
        return fmt.Errorf("prompt build: %w", err)
    }
    return o.harnessAdapter.Dispatch(ctx, task, phase, prompt)
}
```

#### 6.3.4 KillProcess — Interrupção de Harness

Usado quando uma task é deletada durante execução:

```go
func (o *PhaseOrchestrator) KillProcess(taskID string) {
    o.mu.RLock()
    cmd, exists := o.processRegistry[taskID]
    o.mu.RUnlock()
    if exists && cmd.Process != nil {
        cmd.Process.Signal(syscall.SIGKILL)
        o.mu.Lock()
        delete(o.processRegistry, taskID)
        o.mu.Unlock()
    }
}
```

#### 6.3.5 Distinção Crítica: `PhaseOrchestrator.AdvancePhase` vs Use Case `AdvancePhase`

| Método | Quem chama | O que faz |
|--------|-----------|-----------|
| **Use Case `AdvancePhase`** (`advance_phase.go`) | MCP tool `complete_phase` | **Persiste a conclusão da fase**: salva `PhaseOutput`, atualiza `status=completed`, dispara `phase.<phase>.completed`. **Não** inicia a próxima fase. |
| **`PhaseOrchestrator.AdvancePhase`** | Event subscribers (`phase.*.completed`) | **Inicia a próxima fase**: atualiza `current_phase`, reseta `status=pending`, dispara harness para a nova fase. |

Este design em dois passos evita loop infinito: a tool MCP conclui a fase → evento disparado → orchestrator reage e inicia a próxima. O use case nunca chama o orchestrator diretamente, e o orchestrator nunca chama o use case.

#### 6.3.6 Retry Handler

```go
func (o *PhaseOrchestrator) HandleRetry(ctx context.Context, taskID string, phase entity.Phase, attempt int, maxRetries int) {
    if attempt > maxRetries {
        task, _ := o.taskRepo.Find(ctx, taskID)
        task.Status = entity.StatusFailed
        task.UpdatedAt = time.Now()
        _ = o.taskRepo.Update(ctx, task)
        o.dispatcher.Publish(event.Event{
            Type: event.PhaseEvent(phase, "failed"), TaskID: taskID,
            Payload: map[string]any{"phase": phase, "attempt": attempt},
        })
        return
    }
    time.Sleep(time.Duration(2*attempt) * time.Second)
    o.dispatcher.Publish(event.Event{
        Type: event.PhaseEvent(phase, "retry"), TaskID: taskID,
        Payload: map[string]any{"phase": phase, "attempt": attempt},
    })
    task, _ := o.taskRepo.Find(ctx, taskID)
    _ = o.dispatchPhase(ctx, task, phase)
}
```

#### 6.3.7 ReopenPhase — Reabertura para Rework

Quando uma fase downstream (tipicamente **Validating**) detecta problemas que
exigem retrabalho em uma fase **anterior**, ela **não** deve chamar
`complete_phase` e empurrar a falha para frente. Em vez disso, reabre a lane:

```go
func (o *PhaseOrchestrator) ReopenPhase(ctx context.Context, taskID string,
    targetPhase entity.Phase, reason string) error {
    task, err := o.taskRepo.Find(ctx, taskID)
    // ... guards: task ativa; targetPhase não-terminal e anterior à atual ...
    from := task.CurrentPhase
    o.retryUpdate(ctx, taskID, func(t *entity.Task) {
        t.CurrentPhase = targetPhase
        t.Status = entity.StatusPending
        t.ErrorMessage = ""
    })
    o.resetAttempts(taskID)
    o.dispatcher.Publish(event.Event{Type: event.LaneReopened, TaskID: taskID,
        Payload: map[string]any{"from": from, "to": targetPhase, "reason": reason}})
    return o.dispatchPhase(ctx, task, targetPhase)
}
```

**Exposição:**

- **MCP**: tool `reopen_phase` (`task_id`, `phase`=fase atual do harness,
  `target_phase`, `reason`). O `authorize` valida contra a fase atual.
- **HTTP (fallback para harnesses sem MCP, ex.: pi)**:
  `POST /api/v1/tasks/:id/reopen` com body `{ "target_phase": "doing",
  "reason": "..." }`. A base URL é injetada no processo harness via env
  `KANBANAI_API_BASE_URL`.

**Prompt:** todo prompt de fase que pode detectar falha (doing, validating,
testing) recebe um **Failure-Handling Contract** mandatório com a regra de
decisão (complete só em aprovação real; reopen para qualquer falha) e o caminho
HTTP fallback com a URL e `task_id` concretos. Ver `docs/rules.md` §6.

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
│   │   │   ├── task_status.go               # Enum de status (pending, in_progress, completed, failed, cancelled)
│   │   │   ├── task_event_log.go            # Entidade de log de eventos
│   │   │   ├── phase_config.go              # Config de modelo por raia
│   │   │   └── phase_output.go              # Entidade PhaseOutput (NOVO)
│   │   │
│   │   ├── event/
│   │   │   ├── types.go                     # Definição dos EventTypes
│   │   │   ├── event.go                     # Struct Event
│   │   │   ├── dispatcher.go                # Interface Dispatcher
│   │   │   └── handler.go                   # Type Handler
│   │   │
│   │   ├── repository/
│   │   │   ├── task_repository.go           # Interface TaskRepository (com TaskFilters)
│   │   │   ├── task_event_log_repository.go # Interface TaskEventLogRepository
│   │   │   └── phase_output_repository.go   # Interface PhaseOutputRepository (NOVO)
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
│       │   └── bootstrap.go                 # Setup do DI e registro de event subscribers (NOVO)
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
│           │   │   ├── task_repository_sqlite.go       # Impl TaskRepository
│           │   │   ├── task_repository_sqlite_test.go
│           │   │   ├── task_event_log_repository_sqlite.go
│           │   │   ├── task_event_log_repository_sqlite_test.go
│           │   │   ├── phase_output_repository_sqlite.go  # Impl PhaseOutputRepository (NOVO)
│           │   │   └── phase_output_repository_sqlite_test.go
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
    Version      int       // Optimistic locking version
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

### 8.2.1 Status (Enum de Status da Task)

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
    MaxRetries  int       // Quantidade máxima de tentativas automáticas em caso de erro
    TimeoutSec  int       // Limite de execução por tentativa do harness
}
```

### 8.5 PhaseOutput (Entidade de Output da Fase)

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

## 9. Interfaces (Ports)

### 9.1 Repositories (Porta de Saída)

Todos os repositories seguem o contrato padrão com assinaturas estritas para CRUD básico e buscas dinâmicas:

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

### 9.2 Queries Customizadas (Porta de Saída)

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
| `AdvancePhase`          | `advance_phase.go`           | Concluir fase atual (persiste PhaseOutput, dispara phase.*.completed). Não inicia a próxima fase — isso é feito pelo PhaseOrchestrator como subscriber do evento. |
| `ReportPhaseProgress`   | `report_phase_progress.go`   | Registrar progresso de uma fase            |
| `SavePhaseOutput`       | `save_phase_output.go`       | Salvar artefatos/outputs de uma fase       |

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

## 12. SSE (Server-Sent Events)

### 12.1 Broker

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

### 13.1 Protocolo de Transporte MCP e Descoberta

O harness conecta-se ao servidor MCP utilizando um dos dois transportes suportados, configuráveis por variável de ambiente:

1. **Stdio Transport (Padrão para CLI local)**: O `HarnessAdapter` spawna o executável do harness como processo filho. O harness e o KanbanAI se comunicam diretamente via `stdin` e `stdout` redirecionados do processo filho.
2. **SSE Transport (HTTP)**: O servidor MCP abre uma porta HTTP dedicada (`KANBANAI_MCP_PORT=8081`). O `HarnessAdapter` injeta a URL do endpoint SSE (`http://localhost:8081/mcp/sse`) nas variáveis de ambiente do processo filho (`KANBANAI_MCP_URL`). O harness realiza a descoberta se conectando a essa URL.

Para garantir que o harness saiba como se conectar estruturadamente, o `CommandBuilder` escreve um arquivo de configuração temporário no formato padrão do MCP (ex: `mcp_config.json`) na pasta de execução do harness ou injeta variáveis de ambiente:

```json
{
  "mcpServers": {
    "kanbanai-mcp": {
      "command": "kanbanai",
      "args": ["mcp"],
      "env": {
        "KANBANAI_SERVER_URL": "http://localhost:8080"
      }
    }
  }
}
```

### 13.2 Fluxo de Execução com Retries e Timeout

Se o harness falhar em responder dentro do limite definido por `TimeoutSec`, ou retornar um código de erro diferente de zero, o `PhaseOrchestrator` intercepta o erro através do monitoramento do processo e inicia a política de **Retry**:

1. **Backoff Linear**: Um intervalo de espera curto (`2 * tentativa` segundos) é observado antes de re-despachar o comando.
2. **Controle de Tentativas**: O orchestrator incrementa a contagem de tentativas da fase. Se `tentativas > MaxRetries`, o status da task é alterado para `StatusFailed` e o evento `phase.<phase>.failed` é publicado.
3. **Eventos de Retry**: Cada falha transitória dispara um evento `phase.<phase>.retry` para atualizar o frontend via SSE.

### 13.3 Implementação do Adapter

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

    // Executa de forma assíncrona, mas monitora em uma goroutine
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

### 13.4 Command Builder

```go
// internal/adapter/out/harness/command_builder.go
type CommandBuilder struct {
    mcpPort string
}

func (b *CommandBuilder) Build(ctx context.Context, config entity.PhaseConfig, taskID string, prompt string) (*exec.Cmd, error) {
    // Monta o comando injetando KANBANAI_TASK_ID e KANBANAI_MCP_PORT no environment do processo
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

### 14.2 Struct de Configuração

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

## 16. API REST & SSE (End-to-End API Spec)

A API REST do KanbanAI utiliza o prefixo `/api/v1/` e retorna payloads estruturados em JSON. Todos os payloads de sucesso e erro seguem um padrão fixo que o cliente consome utilizando um interceptor HTTP (como Axios/Fetch).

### 16.1 Tabela de Endpoints Completos

| Método   | Rota | Parâmetros de Query / Body | HTTP Status | Descrição |
|----------|------|----------------------------|-------------|-----------|
| `GET`    | `/api/v1/health` | - | `200 OK` | Verificação de integridade da API e conexão do SQLite. |
| `POST`   | `/api/v1/tasks` | **Body**: `{ "title": "string", "description": "string", "priority": int }` | `201 Created` | Registra intenção do usuário, persiste no banco e inicia fluxo Kanban reativo. |
| `GET`    | `/api/v1/tasks` | **Query**: `?phase=string&status=string&limit=10&offset=0` | `200 OK` | Busca tasks com paginação e filtros (mapeados para `Criteria`). |
| `GET`    | `/api/v1/tasks/:id` | **Path**: `id` (ULID) | `200 OK` | Retorna os detalhes de uma task, incluindo os outputs de cada fase já completada. |
| `PUT`    | `/api/v1/tasks/:id` | **Path**: `id`, **Body**: `{ "title": "string", "description": "string", "priority": int, "version": int }` | `200 OK` | Atualização manual de dados pelo usuário (requer correspondência de `version` contra race conditions). |
| `DELETE` | `/api/v1/tasks/:id` | **Path**: `id` | `204 No Content` | Remove a task do banco de dados e sinaliza cancelamento imediato (`SIGKILL`) de harnesses ativos desta task. |
| `GET`    | `/api/v1/tasks/:id/timeline` | **Path**: `id` | `200 OK` | Busca o histórico detalhado de eventos e logs (timeline) associado àquela task específica. |
| `POST`   | `/api/v1/tasks/:id/retry` | **Path**: `id` | `200 OK` | Reinicia manualmente o fluxo de execução para a fase atual travada em estado `failed`. |
| `GET`    | `/api/v1/events` | - | `200 OK` (Stream) | Endpoint SSE. Abre stream persistente e unidirecional para feeds em tempo real do frontend. |

---

### 16.2 Estrutura Detalhada dos payloads (JSON)

#### 1. POST `/api/v1/tasks` (Criar Task)
- **Request Body**:
  ```json
  {
    "title": "Configurar SQLite local",
    "description": "Criar migrations e conexao thread-safe",
    "priority": 2
  }
  ```
- **Response (`201 Created`)**:
  ```json
  {
    "success": true,
    "data": {
      "id": "01J185V1WXP8B4K67R2C8V7Y8E",
      "title": "Configurar SQLite local",
      "description": "Criar migrations e conexao thread-safe",
      "current_phase": "planning",
      "status": "pending",
      "priority": 2,
      "version": 1,
      "created_at": "2026-06-23T21:02:40Z",
      "updated_at": "2026-06-23T21:02:40Z"
    },
    "meta": {
      "request_id": "req-01J185V1Z",
      "timestamp": "2026-06-23T21:02:40Z"
    }
  }
  ```

#### 2. GET `/api/v1/tasks/:id` (Buscar Task com Outputs)
- **Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "task": {
        "id": "01J185V1WXP8B4K67R2C8V7Y8E",
        "title": "Configurar SQLite local",
        "description": "Criar migrations e conexao thread-safe",
        "current_phase": "todo",
        "status": "pending",
        "priority": 2,
        "version": 2,
        "created_at": "2026-06-23T21:02:40Z",
        "updated_at": "2026-06-23T21:05:12Z"
      },
      "phase_outputs": [
        {
          "id": "01J185Y7ZXP8B4K67R2C8V7Y01",
          "task_id": "01J185V1WXP8B4K67R2C8V7Y8E",
          "phase": "planning",
          "output": "# Plano de Execucao\n- Mapear tabelas...\n- Criar conexao...",
          "summary": "Plano de arquitetura e criterios de aceite definidos.",
          "created_at": "2026-06-23T21:05:12Z",
          "updated_at": "2026-06-23T21:05:12Z"
        }
      ]
    },
    "meta": {
      "request_id": "req-01J185Z2X",
      "timestamp": "2026-06-23T21:06:00Z"
    }
  }
  ```

#### 3. Erro Padronizado (ex: Conflito de Concorrência - `409 Conflict`)
- **Response (`409 Conflict`)**:
  ```json
  {
    "success": false,
    "error": {
      "code": "CONCURRENT_MODIFICATION",
      "message": "The task version has changed. Please reload the data and try again."
    },
    "meta": {
      "request_id": "req-01J185Z99",
      "timestamp": "2026-06-23T21:06:05Z"
    }
  }
  ```

---

### 16.3 Mecanismo SSE e Integração com o Frontend (React)

O endpoint `/api/v1/events` mantém uma conexão HTTP persistente aberta (`Connection: keep-alive`, `Content-Type: text/event-stream`).

#### Formato dos Eventos Enviados pelo Go:
O servidor transmite payloads estruturados de acordo com o evento ocorrido. Exemplo de evento de alteração de raia:
```eventstream
event: task.status_changed
data: {"task_id":"01J185V1WXP8B4K67R2C8V7Y8E","title":"Configurar SQLite local","current_phase":"doing","status":"in_progress","version":3}
```

Exemplo de progresso reportado por um agente:
```eventstream
event: phase.doing.progress
data: {"task_id":"01J185V1WXP8B4K67R2C8V7Y8E","phase":"doing","message":"Escrevendo arquivo sqlite_connection.go","progress_percentage":45}
```

#### Como o Frontend consome estes endpoints (React + MUI):
1. **Carregamento Inicial**: Na montagem da página `Dashboard.tsx`, o hook `useTasks.ts` executa um `GET /api/v1/tasks` e popula a renderização das 6 raias do `KanbanBoard`.
2. **Conexão Real-Time**: O hook `useSSE.ts` inicia a conexão com `new EventSource('/api/v1/events')`.
3. **Mutações Reativas**:
   - Ao receber o evento `task.status_changed`, o Contexto React (`TaskContext`) atualiza o estado local movendo o card da task correspondente para a nova coluna (`current_phase`) e aplicando a estilização do status correspondente (`pending`, `in_progress`, `failed`, `completed`).
   - Ao receber `phase.*.progress`, o card correspondente exibe dinamicamente uma barra de progresso linear e um log de atividades em miniatura do agente atrelado.
   - Ao receber `task.created` ou `task.deleted`, o card é inserido ou removido instantaneamente da tela de forma reativa.
   - Em caso de desconexão (ex: queda de rede), o EventSource reconecta automaticamente e atualiza o estado chamando uma nova listagem silenciosa (`GET /api/v1/tasks`).

---

### 16.4 Servindo o Frontend Estático

O servidor Go serve os arquivos estáticos do frontend (build do React) na rota raiz `/`:

```go
// internal/adapter/in/http/router.go
func SetupRoutes(r *gin.Engine, container *di.Container, webDir string) {
    // API v1
    api := r.Group("/api/v1")
    // ... rotas da API ...

    // Servir frontend estático (React build)
    r.Static("/assets", filepath.Join(webDir, "assets"))
    r.StaticFile("/", filepath.Join(webDir, "index.html"))
    r.NoRoute(func(c *gin.Context) {
        c.File(filepath.Join(webDir, "index.html")) // SPA fallback
    })
}
```

O diretório `webDir` é configurável via `KANBANAI_WEB_DIR` (default: `./web`).

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
    version       INTEGER NOT NULL DEFAULT 1, -- Para controle de concorrência via optimistic locking
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

---

## 25. Bootstrap e Fiação de Dependências

O arquivo `internal/adapter/bootstrap/bootstrap.go` é responsável por inicializar todas as dependências do sistema e registrar os observadores (listeners de eventos), garantindo o fluxo reativo sem acoplamento direto:

```go
package bootstrap

import (
    "context"
    "database/sql"
    "log/slog"
    "os"
    "time"

    "kanbanai/config"
    "kanbanai/internal/di"
    "kanbanai/internal/domain/event"
    "kanbanai/internal/adapter/out/event"
    "kanbanai/internal/adapter/out/persistence/sqlite"
    "kanbanai/internal/adapter/out/persistence/repository"
    "kanbanai/internal/adapter/out/persistence/query"
    "kanbanai/internal/adapter/out/harness"
    "kanbanai/internal/adapter/out/sse"
    "kanbanai/internal/application/usecase"
    "kanbanai/internal/application/service"
)

func Initialize(cfg *config.Config) (*di.Container, error) {
    container := di.NewContainer()
    
    // 1. Inicializar Logger (slog)
    var handler slog.Handler
    if cfg.Log.Level == "debug" {
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
    } else {
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
    }
    logger := slog.New(handler)
    slog.SetDefault(logger)
    container.Register("logger", logger)

    // 2. Conectar ao SQLite
    db, err := sqlite.NewConnection(cfg.DB.Path)
    if err != nil {
        return nil, err
    }
    container.Register("db", db)

    // 3. Registrar Repositories e Queries
    taskRepo := repository.NewTaskRepositorySQLite(db)
    eventLogRepo := repository.NewTaskEventLogRepositorySQLite(db)
    phaseOutputRepo := repository.NewPhaseOutputRepositorySQLite(db)
    
    container.Register("taskRepo", taskRepo)
    container.Register("eventLogRepo", eventLogRepo)
    container.Register("phaseOutputRepo", phaseOutputRepo)

    taskWithPhasesQuery := query.NewTaskWithPhasesQuerySQLite(db)
    taskTimelineQuery := query.NewTaskTimelineQuerySQLite(db)
    container.Register("taskWithPhasesQuery", taskWithPhasesQuery)
    container.Register("taskTimelineQuery", taskTimelineQuery)

    // 4. Registrar Dispatcher de Eventos e SSE Broker
    dispatcher := eventimpl.NewDispatcherMemory()
    container.Register("dispatcher", dispatcher)

    sseBroker := sse.NewBroker(dispatcher) // Broker se inscreve via SubscribeAll internamente
    container.Register("sseBroker", sseBroker)

    // 5. Registrar Harness Adapter e Prompt Builder
    promptBuilder := service.NewPromptBuilder()
    container.Register("promptBuilder", promptBuilder)

    harnessAdapter := harness.NewAdapter(cfg.Harness.Phases, cfg.MCP.Port, dispatcher)
    container.Register("harnessAdapter", harnessAdapter)

    // 6. Registrar Use Cases
    createTaskUC := usecase.NewCreateTask(taskRepo, dispatcher)
    updateTaskUC := usecase.NewUpdateTask(taskRepo, dispatcher)
    deleteTaskUC := usecase.NewDeleteTask(taskRepo, dispatcher)
    getTaskUC := usecase.NewGetTask(taskRepo, taskWithPhasesQuery)
    listTasksUC := usecase.NewListTasks(taskRepo)
    advancePhaseUC := usecase.NewAdvancePhase(taskRepo, phaseOutputRepo, dispatcher)
    reportProgressUC := usecase.NewReportPhaseProgress(eventLogRepo, dispatcher)
    savePhaseOutputUC := usecase.NewSavePhaseOutput(phaseOutputRepo, dispatcher)
    
    container.Register("createTaskUseCase", createTaskUC)
    container.Register("updateTaskUseCase", updateTaskUC)
    container.Register("deleteTaskUseCase", deleteTaskUC)
    container.Register("getTaskUseCase", getTaskUC)
    container.Register("listTasksUseCase", listTasksUC)
    container.Register("advancePhaseUseCase", advancePhaseUC)
    container.Register("reportProgressUseCase", reportProgressUC)
    container.Register("savePhaseOutputUseCase", savePhaseOutputUC)

    // 7. Inicializar o PhaseOrchestrator e Registrar Assinaturas do Dispatcher
    orchestrator := service.NewPhaseOrchestrator(
        taskRepo, 
        phaseOutputRepo, 
        harnessAdapter, 
        promptBuilder, 
        dispatcher,
    )
    container.Register("orchestrator", orchestrator)

    // Fiação reativa via Observer Pattern
    // Quando task.created ocorre -> Inicia fluxo do Kanban
    dispatcher.Subscribe(event.TaskCreated, func(evt event.Event) {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        task, err := taskRepo.Find(ctx, evt.TaskID)
        if err != nil {
            logger.Error("bootstrap: failed to find task for StartFlow", "taskID", evt.TaskID, "error", err)
            return
        }
        if err := orchestrator.StartFlow(ctx, task); err != nil {
            logger.Error("bootstrap: StartFlow failed", "taskID", evt.TaskID, "error", err)
        }
    })

    // Quando task.deleted ocorre -> Mata processo harness ativo
    dispatcher.Subscribe(event.TaskDeleted, func(evt event.Event) {
        orchestrator.KillProcess(evt.TaskID)
    })

    // Quando phase.*.completed ocorre -> Avança para a próxima raia
    dispatcher.Subscribe(event.PhasePlanningCompleted, func(evt event.Event) {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        if err := orchestrator.AdvancePhase(ctx, evt.TaskID); err != nil {
            logger.Error("bootstrap: AdvancePhase failed", "taskID", evt.TaskID, "error", err)
        }
    })
    dispatcher.Subscribe(event.PhaseTodoCompleted, func(evt event.Event) {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        if err := orchestrator.AdvancePhase(ctx, evt.TaskID); err != nil {
            logger.Error("bootstrap: AdvancePhase failed", "taskID", evt.TaskID, "error", err)
        }
    })
    dispatcher.Subscribe(event.PhaseDoingCompleted, func(evt event.Event) {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        if err := orchestrator.AdvancePhase(ctx, evt.TaskID); err != nil {
            logger.Error("bootstrap: AdvancePhase failed", "taskID", evt.TaskID, "error", err)
        }
    })
    dispatcher.Subscribe(event.PhaseValidatingCompleted, func(evt event.Event) {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        if err := orchestrator.AdvancePhase(ctx, evt.TaskID); err != nil {
            logger.Error("bootstrap: AdvancePhase failed", "taskID", evt.TaskID, "error", err)
        }
    })
    dispatcher.Subscribe(event.PhaseTestingCompleted, func(evt event.Event) {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        if err := orchestrator.AdvancePhase(ctx, evt.TaskID); err != nil {
            logger.Error("bootstrap: AdvancePhase failed", "taskID", evt.TaskID, "error", err)
        }
    })

    return container, nil
}
```

---

## 26. Controle de Concorrência e Conflitos

Para evitar race conditions decorrentes de múltiplas requisições simultâneas de harnesses ou usuários editando a mesma task, o sistema implementa **Locking Otimista** (Optimistic Locking):

1. A tabela `tasks` possui a coluna `version INTEGER`.
2. Toda query de atualização verifica se a versão enviada pelo use case bate com a versão do banco de dados:
   ```sql
   UPDATE tasks 
   SET title = ?, description = ?, current_phase = ?, status = ?, version = version + 1, updated_at = CURRENT_TIMESTAMP
   WHERE id = ? AND version = ?;
   ```
3. Se a linha afetada for `0`, o repositório retorna um erro `ErrConcurrentModification`.
4. Os Use Cases capturam este erro e efetuam retries automáticos (até 3 tentativas) recarregando a entidade, aplicando a mudança e tentando salvar novamente.

---

## 27. Modelos de Prompts por Fase (PromptBuilder)

O `PromptBuilder` gera prompts direcionados e customizados para cada fase com diretrizes rígidas sobre o uso de ferramentas MCP:

- **Planning**:
  > Você é um arquiteto de software. Analise os requisitos da task "{{.Task.Title}}". Identifique e salve as subtasks e critérios de aceite usando `update_task_output`. Reporte o progresso com `report_progress`. Finalize executando `complete_phase`.
- **Todo**:
  > Você é um Product Owner / Tech Lead. Pegue o planejamento gerado na fase anterior e refine as subtasks em histórias de usuário menores e detalhadas. Atualize os outputs e finalize o refinamento.
- **Doing**:
  > Você é um Engenheiro de Software Sênior. Implemente a solução da task no repositório. Produza código limpo e coeso. Salve o relatório da implementação e os arquivos modificados.
- **Validating**:
  > Você é um Quality Assurance / Reviewer. Analise o código produzido na fase anterior. Realize análises estáticas e valide se todos os critérios de aceite foram atendidos.
- **Testing**:
  > Você é um Engenheiro de Testes de Software. Escreva e execute os testes automatizados unitários/integração para cobrir o código implementado na fase Doing.

---

## 28. Frontend State Management e Integração

O frontend React é gerido por uma arquitetura leve, utilizando Vite e **React Context API** para centralização de estado:

- **TaskContext**: Mantém a lista atualizada de tasks e expõe funções de alteração de estado (`createTask`, `updateTask`).
- **useSSE**: Escuta mensagens do endpoint `/api/v1/events` e despacha mutações diretamente para o `TaskContext` de acordo com o tipo de evento (ex: `task.status_changed`, `phase.doing.progress`), atualizando os cards no board instantaneamente sem necessidade de polling.
- **Vite Config**: O backend URL do Go é repassado dinamicamente via variáveis de ambiente da build (`VITE_API_BASE_URL`).

---

## 29. Graceful Shutdown

O KanbanAI reage de forma graciosa a sinais de terminação do SO (`SIGINT`, `SIGTERM`):

1. **Interrupção de Novos Clientes**: O servidor HTTP/API para de receber novos requests.
2. **Drenagem de Eventos**: O SSE Broker aguarda até 5 segundos para transmitir quaisquer mensagens na fila e encerra as conexões ativas com os navegadores enviando uma terminação limpa.
3. **Cancelamento de Processos Ativos**: O context de aplicação associado aos executáveis do harness é cancelado, terminando processos CLI em andamento.
4. **Fechamento do DB**: A conexão com o banco SQLite é devidamente fechada.

---

## 30. Versionamento da API REST

A API REST do backend segue o padrão de versionamento explícito `/api/v1/`:

- `POST /api/v1/tasks` - Criação de task
- `GET /api/v1/tasks` - Listagem com paginação e filtros
- `GET /api/v1/tasks/:id/timeline` - Timeline de logs da task
- `GET /api/v1/events` - Endpoint SSE

---

## 31. Dockerfile

Uma especificação Docker multi-stage é utilizada para empacotar o executável do backend Go de forma eficiente:

```dockerfile
# Stage 1: Build Go Backend
FROM golang:1.26-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o kanbanai cmd/kanbanai/main.go

# Stage 2: Build Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 3: Final Image
FROM alpine:latest
RUN apk add --no-cache sqlite ca-certificates
WORKDIR /app
COPY --from=backend-builder /app/kanbanai .
COPY --from=frontend-builder /app/frontend/dist ./web
EXPOSE 8080 8081
CMD ["./kanbanai", "serve"]
```

---

## 32. Regras de Fluxo e Ciclo de Vida do Agente (MCP & Orquestração)

Para garantir que o fluxo de execução ocorra de forma previsível e sem travamentos, são adotadas as seguintes regras de governança de ciclo de vida e estado:

### 32.1 Associação e Validação de Task no Servidor MCP
- Sempre que o `HarnessAdapter` spawna um processo filho do harness, ele obrigatoriamente injeta a variável de ambiente `KANBANAI_TASK_ID`.
- O servidor MCP, ao receber requisições de ferramentas como `report_progress`, `update_task_output` ou `complete_phase`, valida se o parâmetro `task_id` fornecido corresponde exatamente ao `KANBANAI_TASK_ID` que o processo foi autorizado a manipular. Caso contrário, a ferramenta retorna imediatamente um erro de segurança/validação para impedir atualizações cruzadas acidentais.

### 32.2 Atualização Dinâmica do Status do Kanban
O ciclo de estados internos do Kanban durante a execução segue estritamente a máquina de estados abaixo:
1. **Transição de Raia**: Quando a fase da task avança (ex: Planning -> Todo), a coluna `current_phase` é atualizada e o `status` da task é imediatamente setado para `pending`.
2. **Sinal de Início**: Assim que o processo do harness correspondente é iniciado ou executa a primeira chamada MCP (seja `report_progress` ou inicialização do protocolo), o `status` da task é movido para `in_progress`.
3. **Sucesso**: Quando a chamada `complete_phase` é executada com sucesso pelo harness, a task entra temporariamente em `status = completed`. O orquestrador reage a este evento e, no mesmo ciclo de transação segura, avança a fase da task resetando para `status = pending` na nova fase (ou finaliza em `status = completed` se a fase alcançada for `Done`).

### 32.3 Monitoramento de Encerramento e Falha do Processo
- A goroutine `monitorProcess` mapeia o ciclo de vida do comando CLI do harness.
- Se o processo CLI do harness terminar com código de saída (exit code) diferente de zero ou estourar o tempo de execução definido por `TimeoutSec`, a goroutine intercepta este encerramento e delega a falha de volta para o orchestrator.
- O orchestrator tenta um retry seguro seguindo o algoritmo de retry (Seção 13.2). Se as tentativas excederem `MaxRetries`, o `status` da task é definitivamente movido para `failed` e a execução do fluxo daquela task específica é bloqueada, necessitando de uma intervenção manual (ex: clique no botão "Restart Phase" no frontend) para limpar o contador de retries e reenviar o evento `started`.
- Se a task for deletada manualmente pelo usuário enquanto o harness estiver rodando, o `PhaseOrchestrator` localiza o processo CLI associado ao `TaskID` e envia um sinal `SIGKILL` (no Windows, mata a árvore de processos correspondente) para liberar recursos e interromper o processamento imediatamente.

