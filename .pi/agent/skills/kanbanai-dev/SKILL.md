---
name: kanbanai-dev
description: This skill should be used when working on the KanbanAI project — adding features, creating use cases, entities, adapters, MCP tools, or any code in this Go + React codebase. It teaches the hexagonal architecture, naming conventions, DI container, event system, and all project-specific rules. Trigger when the user mentions KanbanAI, wants to add a feature, create a use case, add an MCP tool, or implement anything in this project.
version: 0.1.0
---

# KanbanAI Development Guide

This skill teaches how to code in the KanbanAI project following all architectural patterns, naming conventions, and best practices.

## Project Overview

KanbanAI is a Go application that orchestrates an AI-powered Kanban flow. It connects to AI harnesses (Claude, Pi, Hermes) via MCP (Model Context Protocol) to execute each phase autonomously.

**Core flow**: User creates task → persisted in SQLite → events fired via Observer Pattern → frontend notified via SSE → PhaseOrchestrator dispatches harness → harness uses MCP tools to report progress → phases advance automatically until Done.

## Reference Documentation

Before writing any code, read the relevant reference docs from `docs/`:

| Document | When to Read |
|----------|-------------|
| `docs/architecture.md` | Understanding the overall architecture, hexagonal design, SOLID principles, Observer pattern |
| `docs/domain.md` | Creating entities, repository interfaces, query interfaces, ports, or events |
| `docs/application.md` | Creating use cases, DTOs, or modifying the PhaseOrchestrator/PromptBuilder |
| `docs/infrastructure.md` | Creating adapters (HTTP handlers, MCP tools, persistence, harness, SSE), bootstrap wiring |
| `docs/api-spec.md` | Adding/modifying REST endpoints or SSE event formats |
| `docs/frontend.md` | Working on React components, hooks, or state management |
| `docs/configuration.md` | Adding env vars, config structs, or CLI commands |
| `docs/database.md` | Creating migrations or modifying SQLite schema |
| `docs/testing.md` | Writing unit/integration tests with testify mocks |
| `docs/operations.md` | Docker, Makefile, graceful shutdown, performance patterns |
| `docs/conventions.md` | **Always read first** — naming conventions for every element |
| `docs/rules.md` | Business rules: optimistic locking, state machine, retry policy, MCP security |

---

## Naming Conventions (Mandatory)

Every element in this project follows strict naming rules. **Violating these is a bug.**

| Element | Convention | Example |
|---------|-----------|---------|
| Package | `lowercase` singular | `usecase`, `handler`, `entity` |
| Interface | `PascalCase` noun | `TaskRepository`, `Dispatcher` |
| Struct | `PascalCase` noun | `CreateTask`, `SSEBroker` |
| Method | `PascalCase` verb+noun | `Execute`, `FindByFilters` |
| File | `snake_case` | `create_task.go`, `task_handler.go` |
| Test file | `<file>_test.go` | `create_task_test.go` |
| Constant | `PascalCase` with type prefix | `PhasePlanning`, `StatusPending` |
| Env var | `UPPER_SNAKE_CASE` with `KANBANAI_` prefix | `KANBANAI_SERVER_PORT` |
| Event | `dot.notation` lowercase | `task.created`, `phase.planning.started` |
| DTO | `PascalCase` + Input/Output/Filter | `CreateTaskInput`, `TaskOutput` |
| Repository impl | Interface + `SQLite` suffix | `TaskRepositorySQLite` |
| Query impl | Interface + `SQLite` suffix | `TaskWithPhasesQuerySQLite` |
| Mock | `Mock` + Interface name | `MockTaskRepository` |

---

## Architecture Rules

### Hexagonal (Ports & Adapters)

```
Adapters IN (HTTP/CLI/MCP) → Use Cases → Domain (Entities + Interfaces) ← Adapters OUT (SQLite/Harness/SSE/Event)
```

- **Domain** defines interfaces (`repository/`, `query/`, `port/`)
- **Application** implements use cases and services, depending ONLY on domain interfaces
- **Adapters** implement domain interfaces (IN: HTTP handlers, MCP tools; OUT: SQLite repos, harness client, SSE broker)
- **NEVER** import an adapter from domain or application layer
- **NEVER** use `new` directly in use cases — always receive dependencies via constructor

### DI Container

All dependencies are registered in `internal/adapter/bootstrap/bootstrap.go` and resolved via `internal/di/container.go`:

```go
// Registering
container.Register("taskRepo", taskRepo)

// Resolving
taskRepo := container.MustResolve("taskRepo").(repository.TaskRepository)
```

Registration names follow camelCase: `taskRepo`, `createTaskUseCase`, `harnessAdapter`, `orchestrator`.

### Observer Pattern (Events)

Events are the backbone of reactivity. Components communicate ONLY through events, never directly.

```go
// Publishing
dispatcher.Publish(event.Event{
    Type:   event.TaskCreated,
    TaskID: task.ID,
    Payload: map[string]any{"phase": task.CurrentPhase},
})

// Subscribing (in bootstrap.go)
dispatcher.Subscribe(event.TaskCreated, func(evt event.Event) {
    // react to event
})
```

The SSE Broker uses `SubscribeAll` (wildcard) to forward all events to connected frontend clients.

---

## How to Add a New Feature

### Step 1: Identify the Layer

| What you're adding | Where it goes |
|--------------------|---------------|
| New domain concept (struct) | `internal/domain/entity/` |
| New event type | `internal/domain/event/types.go` |
| New repository interface | `internal/domain/repository/` |
| New query interface (joins) | `internal/domain/query/` |
| New port interface | `internal/domain/port/` |
| New use case | `internal/application/usecase/` |
| New DTO | `internal/application/dto/` |
| New application service | `internal/application/service/` |
| New HTTP handler | `internal/adapter/in/http/handler/` |
| New MCP tool | `internal/adapter/in/mcp/` |
| New CLI command | `internal/adapter/in/cli/` |
| New SQLite repository impl | `internal/adapter/out/persistence/repository/` |
| New SQLite query impl | `internal/adapter/out/persistence/query/` |
| New migration | `internal/adapter/out/persistence/sqlite/migration_files/` |

### Step 2: Follow the Dependency Chain

Always work **outside-in** or **inside-out** consistently:

**Inside-Out (Domain First)**:
1. Create entity in `internal/domain/entity/`
2. Create repository/query/port interface in `internal/domain/`
3. Create DTO in `internal/application/dto/`
4. Create use case in `internal/application/usecase/`
5. Create adapter implementation in `internal/adapter/`
6. Register in `internal/adapter/bootstrap/bootstrap.go`
7. Wire event subscriptions if needed
8. Write tests

### Step 3: Use Case Template

```go
// internal/application/usecase/my_new_use_case.go
package usecase

type MyNewUseCase struct {
    taskRepo   repository.TaskRepository
    dispatcher event.Dispatcher
}

func NewMyNewUseCase(repo repository.TaskRepository, disp event.Dispatcher) *MyNewUseCase {
    return &MyNewUseCase{taskRepo: repo, dispatcher: disp}
}

func (uc *MyNewUseCase) Execute(ctx context.Context, input dto.MyInput) (*dto.MyOutput, error) {
    // 1. Validate input
    // 2. Operate on domain entities
    // 3. Persist via repository
    // 4. Publish events via dispatcher
    // 5. Return DTO
}
```

### Step 4: Repository Implementation Template

```go
// internal/adapter/out/persistence/repository/my_repository_sqlite.go
package repository

type MyRepositorySQLite struct {
    db *sql.DB
}

func NewMyRepositorySQLite(db *sql.DB) *MyRepositorySQLite {
    return &MyRepositorySQLite{db: db}
}

func (r *MyRepositorySQLite) Create(ctx context.Context, entity *entity.MyEntity) error {
    // Use prepared statements
    // Handle optimistic locking if applicable
}
```

### Step 5: HTTP Handler Template

```go
// internal/adapter/in/http/handler/my_handler.go
package handler

type MyHandler struct {
    myUseCase *usecase.MyNewUseCase
}

func NewMyHandler(container *di.Container) *MyHandler {
    return &MyHandler{
        myUseCase: container.MustResolve("myNewUseCase").(*usecase.MyNewUseCase),
    }
}

func (h *MyHandler) Handle(c *gin.Context) {
    // Parse request
    // Call use case
    // Return standardized response
}
```

### Step 6: Bootstrap Registration

In `internal/adapter/bootstrap/bootstrap.go`:

```go
// Register implementation
myRepo := repository.NewMyRepositorySQLite(db)
container.Register("myRepo", myRepo)

// Register use case
myUseCase := usecase.NewMyNewUseCase(myRepo, dispatcher)
container.Register("myNewUseCase", myUseCase)

// Wire events if needed
dispatcher.Subscribe(event.SomeEvent, func(evt event.Event) {
    // ...
})
```

### Step 7: Test Template

```go
// internal/application/usecase/my_new_use_case_test.go
package usecase_test

func TestMyNewUseCase_Execute_Success(t *testing.T) {
    mockRepo := new(mocks.MockTaskRepository)
    mockDispatcher := new(mocks.MockDispatcher)
    
    mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Task")).Return(nil)
    mockDispatcher.On("Publish", mock.AnythingOfType("event.Event")).Return()
    
    uc := usecase.NewMyNewUseCase(mockRepo, mockDispatcher)
    result, err := uc.Execute(context.Background(), input)
    
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
    mockDispatcher.AssertExpectations(t)
}
```

---

## Critical Rules

### 1. AdvancePhase Distinction

There are TWO different "AdvancePhase" concepts — **never confuse them**:

- **Use Case `AdvancePhase`** (`advance_phase.go`): Called by MCP tool `complete_phase`. Persists phase completion, fires `phase.*.completed`. Does NOT start next phase.
- **`PhaseOrchestrator.AdvancePhase`**: Subscribes to `phase.*.completed`. Starts the next phase by dispatching harness.

The use case never calls the orchestrator. The orchestrator never calls the use case. They communicate only through events.

### 2. Status State Machine

```
pending → in_progress → completed → pending (next phase)
                   ↘ failed (terminal)
pending → cancelled (terminal)
```

### 3. Optimistic Locking

Every UPDATE on tasks MUST check `version`:
```sql
UPDATE tasks SET ... version = version + 1 WHERE id = ? AND version = ?;
```
If 0 rows affected, return `ErrConcurrentModification`. Use cases retry up to 3 times.

### 4. MCP Security

Every MCP tool MUST validate that `task_id` argument matches the process's `KANBANAI_TASK_ID` env var. Reject mismatches immediately.

### 5. Error Handling

Always wrap errors with context:
```go
if err != nil {
    return fmt.Errorf("failed to create task: %w", err)
}
```

### 6. Context Propagation

Every method that does I/O MUST receive `context.Context` as first parameter.

---

## Quick Reference: Key Files

| File | Purpose |
|------|---------|
| `cmd/kanbanai/main.go` | Entrypoint |
| `internal/di/container.go` | DI container |
| `internal/adapter/bootstrap/bootstrap.go` | All wiring, DI registration, event subscriptions |
| `internal/domain/event/types.go` | All event type constants |
| `internal/domain/entity/task.go` | Core Task entity |
| `internal/domain/entity/task_phase.go` | Phase enum with `Next()` and `IsTerminal()` |
| `internal/domain/entity/task_status.go` | Status enum |
| `internal/application/service/phase_orchestrator.go` | Central orchestrator |
| `internal/application/service/prompt_builder.go` | Generates prompts per phase |
| `internal/adapter/out/harness/adapter.go` | Spawns harness CLI processes |
| `internal/adapter/out/sse/broker.go` | SSE connection hub |
| `internal/adapter/in/mcp/tools.go` | MCP tool registration |
| `internal/adapter/in/http/router.go` | Route definitions |
| `config/config.go` | Configuration struct |
