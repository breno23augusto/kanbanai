# Naming Conventions

Every element in this project follows strict naming rules. **Violating these is a bug.**

## Package, File, and Symbol Naming

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

## DTO, Repository, and Mock Naming

| Element | Convention | Example |
|---------|-----------|---------|
| DTO | `PascalCase` + Input/Output/Filter | `CreateTaskInput`, `TaskOutput` |
| Repository impl | Interface + `SQLite` suffix | `TaskRepositorySQLite` |
| Query impl | Interface + `SQLite` suffix | `TaskWithPhasesQuerySQLite` |
| Mock | `Mock` + Interface name | `MockTaskRepository` |

## DI Registration Names

Registration names follow camelCase:

```
taskRepo, createTaskUseCase, harnessAdapter, orchestrator
```

## Event Type Constants

Defined in `internal/domain/event/types.go`:

```go
const (
    TaskCreated        = "task.created"
    TaskStatusChanged  = "task.status.changed"
    PhasePlanningStarted   = "phase.planning.started"
    PhasePlanningCompleted = "phase.planning.completed"
    // ... per phase: .started and .completed
)
```
