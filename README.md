# KanbanAI

AI-powered Kanban orchestrator. Connects to AI harnesses (Claude, Pi, Hermes) via MCP to execute each phase autonomously.

## Architecture

```
User → Frontend (React + MUI) → HTTP API (Gin) → Use Cases → Domain → Adapters (SQLite, Harness, SSE)
```

### Kanban Lanes

1. **Planning** — Task scope, subtasks, acceptance criteria
2. **Todo** — Refined backlog ready for execution
3. **Doing** — Active implementation by harness
4. **Validating** — Code review and requirements validation
5. **Testing** — Automated and manual tests
6. **Done** — Task completed

## Quick Start

```bash
# Copy environment config
cp .env.example .env

# Run database migrations
make migrate

# Start the server
make dev
```

## Commands

| Command | Description |
|---------|-------------|
| `kanbanai serve` | Start HTTP + MCP servers |
| `kanbanai migrate` | Run database migrations |
| `kanbanai version` | Show version |

## API Endpoints

| Method | Route | Description |
|--------|-------|-------------|
| `GET` | `/api/v1/health` | Health check |
| `POST` | `/api/v1/tasks` | Create task |
| `GET` | `/api/v1/tasks` | List tasks |
| `GET` | `/api/v1/tasks/:id` | Get task details |
| `PUT` | `/api/v1/tasks/:id` | Update task |
| `DELETE` | `/api/v1/tasks/:id` | Delete task |
| `GET` | `/api/v1/tasks/:id/timeline` | Task event timeline |
| `POST` | `/api/v1/tasks/:id/retry` | Retry failed phase |
| `GET` | `/api/v1/events` | SSE event stream |

## MCP Tools

| Tool | Description |
|------|-------------|
| `get_task` | Retrieve task information |
| `report_progress` | Report phase progress |
| `update_task_output` | Save phase artifacts |
| `complete_phase` | Complete current phase |

## Tech Stack

- **Language**: Go 1.26
- **HTTP**: Gin
- **MCP**: modelcontextprotocol/go-sdk
- **CLI**: Cobra
- **Config**: Viper
- **Database**: SQLite (mattn/go-sqlite3)
- **Frontend**: React 18 + MUI 5
- **Real-time**: SSE
