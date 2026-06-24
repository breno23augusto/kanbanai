---
name: kanbanai-dev
description: |
  This skill should be used when working on the KanbanAI project — adding features, creating use cases, entities, adapters, MCP tools, or any code in this Go + React codebase. Covers hexagonal architecture, naming conventions, DI container, event system, and all project-specific rules.
triggers:
  - "KanbanAI"
  - "add a feature to KanbanAI"
  - "create a use case"
  - "add an MCP tool"
  - "KanbanAI project"
  - "kanbanai"
  - "implement in KanbanAI"
  - "KanbanAI codebase"
version: 0.2.0
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

## Architecture Summary

### Hexagonal (Ports & Adapters)

```
Adapters IN (HTTP/CLI/MCP) → Use Cases → Domain (Entities + Interfaces) ← Adapters OUT (SQLite/Harness/SSE/Event)
```

- **Domain** defines interfaces (`repository/`, `query/`, `port/`)
- **Application** implements use cases and services, depending ONLY on domain interfaces
- **Adapters** implement domain interfaces
- **NEVER** import an adapter from domain or application layer
- **NEVER** use `new` directly in use cases — always receive dependencies via constructor

### DI Container

All dependencies are registered in `internal/adapter/bootstrap/bootstrap.go` and resolved via `internal/di/container.go`. Registration names follow camelCase: `taskRepo`, `createTaskUseCase`, `harnessAdapter`, `orchestrator`.

### Observer Pattern (Events)

Components communicate ONLY through events, never directly. The SSE Broker uses `SubscribeAll` (wildcard) to forward all events to connected frontend clients.

For detailed architecture rules (AdvancePhase distinction, state machine, optimistic locking, MCP security, error handling), see **`references/architecture-rules.md`**.

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

Work **inside-out** (Domain First):

1. Create entity in `internal/domain/entity/`
2. Create repository/query/port interface in `internal/domain/`
3. Create DTO in `internal/application/dto/`
4. Create use case in `internal/application/usecase/`
5. Create adapter implementation in `internal/adapter/`
6. Register in `internal/adapter/bootstrap/bootstrap.go`
7. Wire event subscriptions if needed
8. Write tests

### Step 3: Use the Templates

For code templates (use case, repository, handler, bootstrap registration, tests), see **`references/templates.md`**.

---

## Critical Rules (Summary)

1. **AdvancePhase Distinction**: The use case `AdvancePhase` persists completion and fires events. The `PhaseOrchestrator.AdvancePhase` subscribes to those events and starts the next phase. They communicate only through events — never call each other directly.

2. **Status State Machine**: `pending → in_progress → completed → pending (next phase)`. `failed` and `cancelled` are terminal.

3. **Optimistic Locking**: Every UPDATE checks `version`. Retry up to 3 times on `ErrConcurrentModification`.

4. **MCP Security**: Every MCP tool validates `task_id` against `KANBANAI_TASK_ID` env var.

5. **Error Handling**: Always wrap with `fmt.Errorf("context: %w", err)`.

6. **Context Propagation**: Every I/O method receives `context.Context` as first parameter.

Full details in **`references/architecture-rules.md`**.

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

---

## Additional Resources

### Reference Files (in this skill)

- **`references/naming-conventions.md`** — Full naming rules for packages, files, symbols, DTOs, repos, mocks, DI names, and event constants
- **`references/architecture-rules.md`** — Detailed rules: hexagonal design, DI container, Observer pattern, AdvancePhase distinction, state machine, optimistic locking, MCP security, error handling, context propagation
- **`references/templates.md`** — Copy-paste templates for use cases, repository implementations, HTTP handlers, bootstrap registration, and tests

### Project Docs

All detailed documentation lives in `docs/` at the project root. Consult the table above to find the right doc for each task.
