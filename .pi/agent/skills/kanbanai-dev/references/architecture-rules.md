# Architecture Rules

## Hexagonal (Ports & Adapters)

```
Adapters IN (HTTP/CLI/MCP) → Use Cases → Domain (Entities + Interfaces) ← Adapters OUT (SQLite/Harness/SSE/Event)
```

- **Domain** defines interfaces (`repository/`, `query/`, `port/`)
- **Application** implements use cases and services, depending ONLY on domain interfaces
- **Adapters** implement domain interfaces (IN: HTTP handlers, MCP tools; OUT: SQLite repos, harness client, SSE broker)
- **NEVER** import an adapter from domain or application layer
- **NEVER** use `new` directly in use cases — always receive dependencies via constructor

## DI Container

All dependencies are registered in `internal/adapter/bootstrap/bootstrap.go` and resolved via `internal/di/container.go`:

```go
// Registering
container.Register("taskRepo", taskRepo)

// Resolving
taskRepo := container.MustResolve("taskRepo").(repository.TaskRepository)
```

## Observer Pattern (Events)

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

## AdvancePhase Distinction

There are TWO different "AdvancePhase" concepts — **never confuse them**:

- **Use Case `AdvancePhase`** (`advance_phase.go`): Called by MCP tool `complete_phase`. Persists phase completion, fires `phase.*.completed`. Does NOT start next phase.
- **`PhaseOrchestrator.AdvancePhase`**: Subscribes to `phase.*.completed`. Starts the next phase by dispatching harness.

The use case never calls the orchestrator. The orchestrator never calls the use case. They communicate only through events.

## Status State Machine

```
pending → in_progress → completed → pending (next phase)
                   ↘ failed (terminal)
pending → cancelled (terminal)
```

## Optimistic Locking

Every UPDATE on tasks MUST check `version`:

```sql
UPDATE tasks SET ... version = version + 1 WHERE id = ? AND version = ?;
```

If 0 rows affected, return `ErrConcurrentModification`. Use cases retry up to 3 times.

## MCP Security

Every MCP tool MUST validate that `task_id` argument matches the process's `KANBANAI_TASK_ID` env var. Reject mismatches immediately.

## Error Handling

Always wrap errors with context:

```go
if err != nil {
    return fmt.Errorf("failed to create task: %w", err)
}
```

## Context Propagation

Every method that does I/O MUST receive `context.Context` as first parameter.
