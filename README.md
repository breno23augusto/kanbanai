# KanbanAI

An AI-powered Kanban board where cards move themselves through the pipeline.
Create a task, and an autonomous coding agent (the **harness**) picks it up,
breaks it into subtasks, writes the implementation, reviews it against the
original requirements, tests it, and advances the card lane-by-lane — calling
back into the board over MCP to report progress, persist output, and finalize
each phase. You watch it happen live in the browser.

```
 You create a task ──► planning ──► todo ──► doing ──► validating ──► testing ──► done
                          │           │         │            │            │
                          └───────────┴── an AI harness runs each lane ───┘
                                       and drives the board via MCP tools
```

The board is the source of truth; the agent is a worker that talks to it. If a
phase fails review, the card is sent back a lane (reopened) with the reviewer's
feedback injected into the next attempt.

---

## Table of contents

1. [What you get](#what-you-get)
2. [Architecture](#architecture)
3. [Prerequisites](#prerequisites)
4. [Quick start (dev)](#quick-start-dev)
5. [Using the dashboard](#using-the-dashboard)
6. [How a task flows (the agent loop)](#how-a-task-flows-the-agent-loop)
7. [Harnesses](#harnesses)
   - [pi harness (default)](#pi-harness-default)
   - [Claude Code harness](#claude-code-harness)
8. [Configuration](#configuration)
   - [Environment variables](#environment-variables)
   - [Per-lane config (the gear button)](#per-lane-config-the-gear-button)
9. [REST API](#rest-api)
10. [MCP tools](#mcp-tools)
11. [Project structure](#project-structure)
12. [Development](#development)
13. [Operations (systemd)](#operations-systemd)
14. [Troubleshooting](#troubleshooting)

---

## What you get

- **A Kanban board** (React + MUI) with 6 lanes: `planning → todo → doing → validating → testing → done`.
- **Autonomous phase execution** — each lane is run by an AI harness that receives a phase-specific prompt and is expected to do the work, then finalize the phase by calling back into the board.
- **Subtasks** — the planning agent breaks the task into tracked subtasks; the doing agent advances each one. Per-subtask status shows on the card and in the drawer.
- **Validation with feedback loop** — the validating agent compares the *original prompt* against *what was actually implemented* and either advances the card or reopens it to `doing` with the reason injected into the next attempt.
- **Live tail** — watch the agent's reasoning, text, and tool calls stream into the drawer in real time while it works.
- **Per-lane configuration** — set the harness command, model, retries, and timeout per lane from the UI (a gear button); changes take effect on the next dispatch, no restart.
- **Per-task workspace** — each task can specify the directory the harness runs in (overrides the server default).
- **Two harnesses supported** out of the box: the **pi** coding agent (default, SDK-based) and **Claude Code** (MCP-native).

---

## Architecture

```
Browser ──HTTP/SSE──► Gin HTTP API ──► Use Cases ──► Domain ──► Adapters
                         │                                        ├─ SQLite (persistence)
        MCP SSE (:18401) │                                        ├─ Harness (spawns agent process)
                         │                                        └─ LiveTail (in-memory ring buffer)
                         └── also serves the built React frontend (web/)

Agent process (pi/claude) ──MCP──► KanbanAI MCP server (:18401)
   called by the harness         └─ 7 tools: get_task, report_progress,
                                   update_task_output, complete_phase,
                                   reopen_phase, create_subtasks,
                                   update_subtask_status
```

Hexagonal architecture: the **domain** (entity/usecase) is pure Go; **adapters**
(SQLite, HTTP, MCP, harness, livetail) plug in at the edges via interfaces and
are wired together in `internal/adapter/bootstrap`. The DI container
(`internal/di`) resolves everything at boot.

- **HTTP** (Gin): the API + serves the built frontend. Port `18400` in dev (default `8080`).
- **MCP** (modelcontextprotocol/go-sdk, SSE transport): a dedicated port `18401` (default `8081`). Agents connect here.
- **Persistence**: SQLite at `./data/kanbanai.db`, migrations in `internal/adapter/out/persistence/sqlite/migration_files/`.

---

## Prerequisites

- **Go** 1.26+ (CGO enabled — uses `mattn/go-sqlite3`)
- **Node.js** 20+ and **npm** (for the frontend)
- **An AI harness** — at least one of:
  - the **pi** coding agent (`@earendil-works/pi-coding-agent` installed globally, plus its typebox dep), **or**
  - **Claude Code** (`claude` on `PATH`)
- The harness needs a backing model provider (e.g. an Ollama instance, or an Anthropic API key for Claude).

The dev setup in this repo uses pi + an Ollama-served `deepseek-v4-flash:cloud` model.

---

## Quick start (dev)

The repo ships a ready-to-run dev config (`.env.dev`) that uses the **pi harness** with `deepseek-v4-flash:cloud`.

```bash
# 1. Install Go deps + the pi harness (global, for the SDK wrapper)
go mod download
npm install -g @earendil-works/pi-coding-agent

# 2. Build the frontend (served by the Go binary at /)
make frontend-build

# 3. Build the server
make build

# 4. Copy + edit env (the dev file pins pi + an Ollama model)
cp .env.example .env          # then edit: set ports, harness cmd, model
#    (or just use the provided .env.dev — see "Operations" below)

# 5. Run migrations (creates ./data/kanbanai.db)
make migrate

# 6. Start the server
make dev          # go run cmd/kanbanai/main.go serve
```

Then open **http://localhost:18400** (or whatever `KANBANAI_SERVER_PORT` you set).

> The dev config (`.env.dev`) points the harness at `scripts/pi-harness.sh` and
> model `ollama/deepseek-v4-flash:cloud`. If your model provider differs, edit
> `KANBANAI_HARNESS_DEFAULT_MODEL` (or override per-lane — see below).

---

## Using the dashboard

1. **Create a task** — the **+** button (top bar) opens a dialog. Give it a
   title, a description (this is the *original prompt* the validator later
   checks against), an optional priority, and an optional **workspace** (the
   directory the harness will run in for this task; empty = server default
   `PI_HARNESS_CWD`).
2. **Watch it flow** — the new card lands in `planning`. A harness dispatches
   automatically, the card flips to `in_progress`, and the **● LIVE** indicator
   lights up. Click the card to open the detail drawer.
3. **The detail drawer** shows:
   - metadata (phase, status, workspace, priority, version)
   - the **rework feedback** block (amber) when the card was reopened — the
     reviewer's reason + downstream review, injected into the next attempt
   - the **subtask checklist** (created in planning, advanced in doing) with
     per-item status, strikethrough for completed, highlight for in-progress
   - the **live tail** panel — the agent's reasoning, prose, and tool calls
     streaming in real time (only while `in_progress`)
   - the phase **output** markdown once finalized
   - the **event timeline**
4. **Buttons in the drawer action bar**:
   - **Retry** — re-run a failed phase
   - **Pause / Resume** — stop the running harness without losing state, resume later
   - **Delete** — remove the task (with confirmation)
5. **Per-lane config (⚙ gear, top bar)** — a dialog with one row per lane
   (`planning`…`testing`) for `model`, `harness cmd`, `max retries`, and
   `timeout`. Empty a field to inherit the env default. **Reset per-lane** or
   **reset all**. Changes apply to the next dispatch immediately (no restart) and
   survive server restarts (persisted to the DB).
6. The board auto-updates over a global **SSE** stream (`/api/v1/events`) — no
   polling. New tasks, phase moves, subtask changes, and config updates all
   push live.

---

## How a task flows (the agent loop)

For each phase a task enters, the orchestrator:

1. Builds a **phase-specific prompt** (`PromptBuilder`) containing the task
   title/description, the task ID, the MCP base URL, the subtask checklist
   (for doing/validating/testing), any prior-phase outputs (for context), and —
   on rework — the reopen reason + downstream review feedback.
2. Asks the `PhaseConfigProvider` for that lane's effective config (env default
   merged with any DB override).
3. Spawns the **harness** (`<cmd> --model <model> --prompt <prompt>`) with the
   task ID, phase, MCP URL, API base URL, and (if set) the workspace, in its
   environment. The harness's stdout/stderr stream to the live tail.
4. The **agent** does the work and finalizes the phase itself by calling MCP
   tools: `create_subtasks` (planning), `update_subtask_status` (doing),
   `update_task_output` (persist artifacts), and `complete_phase` (advance) or
   `reopen_phase` (send back with a reason).
5. **Safety net**: if the agent exits without finalizing, the harness wrapper
   persists whatever output it captured and auto-completes the phase — so a
   lane never hangs. (Tolerant: if the phase already advanced by any path, it
   skips.)

Phase order: `planning → todo → doing → validating → testing → done`.
`complete_phase` advances forward; `reopen_phase` sends the card back one lane
(e.g. `validating → doing`) with a reason. The reopen reason is cleared on the
next forward advance so stale feedback doesn't leak into later cycles.

---

## Harnesses

A *harness* is the script KanbanAI spawns to run an AI agent for a lane. It's
invoked as `<cmd> --model <model> --prompt <prompt>` and receives
`KANBANAI_TASK_ID`, `KANBANAI_PHASE`, `KANBANAI_MCP_URL`,
`KANBANAI_API_BASE_URL` (and `PI_HARNESS_CWD` if a workspace is set) in its
environment. The harness is responsible for bridging the agent to the board.

### pi harness (default)

`scripts/pi-harness.sh` → `scripts/pi-harness.mjs`

The pi coding agent has **no MCP support**, so this wrapper registers KanbanAI's
operations as pi `customTools` (via `defineTool` + TypeBox schemas) that call
the REST API under the hood — the names match the MCP tool names, so prompts
read the same regardless of transport. It streams `text_delta` (prose),
`thinking_delta` (reasoning), and `toolcall_end` markers to the live tail via
synchronous `fs.writeSync(2, …)` (Node's async stderr would buffer). The
persisted phase output accumulates only assistant prose (not thinking) to stay
clean.

Requires: `@earendil-works/pi-coding-agent` globally installed and
`PI_PKG_DIR` resolvable (the launcher sets it).

### Claude Code harness

`scripts/claude-harness.sh` → `scripts/claude-harness.mjs`

Claude Code **speaks MCP natively**, so this wrapper just spawns
`claude -p --output-format stream-json --verbose` with an MCP config pointing
at KanbanAI's SSE server — the agent calls the real MCP tools directly, no REST
shim. It pretty-prints stream-json events (assistant text/thinking, tool_use,
tool_result) to the live tail, and has the same safety-net as the pi harness.

**Headless auth**: Claude Code normally authenticates via the interactive
keyring, which a background server can't reach. To run it headless, put your
`ANTHROPIC_*` env vars in a gitignored `.env.local` (sourced by
`scripts/run-kanbanai.sh` if present):

```bash
# .env.local (gitignored, chmod 600)
export ANTHROPIC_BASE_URL=https://api.anthropic.com     # or your provider
export ANTHROPIC_AUTH_TOKEN=sk-ant-...
export ANTHROPIC_DEFAULT_SONNET_MODEL=claude-sonnet-4-5
export ANTHROPIC_DEFAULT_OPUS_MODEL=...
export ANTHROPIC_DEFAULT_HAIKU_MODEL=...
```

To switch a lane to Claude, open the **gear** and set that lane's
`harness cmd` to the absolute path of `scripts/claude-harness.sh` and its
`model` to a Claude-recognized alias (`sonnet`/`opus`/`haiku`) or full id. The
default stays pi unless you change it.

> **Model matters for agentic quality.** MCP tool-use is demanding. A weak model
> may refuse to finalize a phase; the safety-net will still advance the lane, but
> for clean autonomous runs use a capable model on the lanes that matter.

---

## Configuration

### Environment variables

All read by `config/loader.go` (Viper). `KANBANAI_*` only.

| Variable | Default | Purpose |
|---|---|---|
| `KANBANAI_SERVER_PORT` | `8080` | HTTP API + frontend port (dev: `18400`) |
| `KANBANAI_SERVER_HOST` | `0.0.0.0` | HTTP bind host |
| `KANBANAI_DB_PATH` | `./data/kanbanai.db` | SQLite path |
| `KANBANAI_DB_MIGRATION_DIR` | `…/migration_files` | Embedded SQL migrations |
| `KANBANAI_MCP_PORT` | `8081` | MCP SSE port (dev: `18401`) |
| `KANBANAI_WEB_DIR` | `./web` | Built frontend served at `/` |
| `KANBANAI_LOG_LEVEL` | `info` | Log level |
| `KANBANAI_HARNESS_DEFAULT_CMD` | — | Harness command for any lane without an override |
| `KANBANAI_HARNESS_DEFAULT_MODEL` | — | Model for any lane without an override |
| `KANBANAI_HARNESS_MAX_RETRIES` | `3` | Global retry count (backoff = 2×attempt s) |
| `KANBANAI_HARNESS_TIMEOUT_SEC` | `600` | Global per-attempt timeout (seconds) |
| `KANBANAI_HARNESS_<PHASE>_{CMD,MODEL,MAX_RETRIES,TIMEOUT_SEC}` | — | Per-lane override (`<PHASE>` = `PLANNING`/`TODO`/`DOING`/`VALIDATING`/`TESTING`) |

Per-lane env vars override the `DEFAULT_*` ones; empty/missing falls back to the
default. The **UI gear** writes DB overrides on top of these (see below), so
you usually don't need per-lane env vars at all.

Harness-wrapper support vars (inherited by the spawned agent process):

| Variable | Purpose |
|---|---|
| `KANBANAI_API_URL` / `KANBANAI_API_BASE_URL` | Base URL the wrapper uses to call back (set by the launcher/CommandBuilder) |
| `PI_HARNESS_TOOLS` | CSV of pi tool names to enable (e.g. `read,bash,grep,ls,write,edit`); empty = pure reasoning |
| `PI_HARNESS_CWD` | Working dir for the agent; overridden per-task by the task's `workspace` |
| `PI_PKG_DIR` | Path to the globally-installed pi SDK package (set by `run-kanbanai.sh`) |

### Per-lane config (the gear button)

The **⚙** button in the top bar opens `PhaseConfigDialog` — one row per
non-terminal lane for `model`, `harness cmd`, `max retries`, `timeout`. The
dialog shows each field's **default** (from env) as a placeholder; typing a
value creates a DB override, leaving it empty inherits the default. **Reset**
clears overrides. Overrides are read on every dispatch (`provider.Get(phase)`),
so a change takes effect immediately on the next task that enters that lane,
and they survive restarts (loaded at boot via `provider.Reload()`).

Equivalent REST: `GET/PUT /api/v1/config/phases`.

---

## REST API

Base: `http://<host>:<port>/api/v1`

| Method | Route | Description |
|---|---|---|
| `GET` | `/health` | Health check |
| `POST` | `/tasks` | Create task `{title, description, priority, workspace?}` |
| `GET` | `/tasks` | List tasks (with phases, subtasks, summary) |
| `GET` | `/tasks/:id` | Get task detail (task + subtasks + phase outputs) |
| `PUT` | `/tasks/:id` | Update task `{title, description, priority, workspace?, version}` |
| `DELETE` | `/tasks/:id` | Delete task |
| `GET` | `/tasks/:id/timeline` | Event timeline |
| `POST` | `/tasks/:id/retry` | Retry a failed phase |
| `POST` | `/tasks/:id/pause` | Pause the running harness |
| `POST` | `/tasks/:id/resume` | Resume a paused task |
| `POST` | `/tasks/:id/complete` | Advance current phase (complete_phase) |
| `POST` | `/tasks/:id/reopen` | Reopen to previous phase with `{reason}` |
| `POST` | `/tasks/:id/subtasks` | Replace subtask list `{subtasks:[{title}]}` |
| `PATCH` | `/tasks/:id/subtasks/:sid` | Update subtask status `{status}` |
| `PUT` | `/tasks/:id/output` | Save phase raw output `{output}` |
| `GET` | `/tasks/:id/live` | **SSE** — live harness output stream (replays last 64KB) |
| `GET` | `/events` | **SSE** — global board event stream |
| `GET` | `/config/phases` | Effective + default per-lane config |
| `PUT` | `/config/phases` | Replace per-lane config overrides |

---

## MCP tools

Exposed over SSE at `http://<host>:<mcp_port>/mcp/sse`. The harness passes the
task ID and current phase to each write tool; the server's `authorize` layer
validates that the task exists, is in an active status, and that its
`current_phase` matches — so an agent can only mutate the task whose lane it's
currently running.

| Tool | Phase | Description |
|---|---|---|
| `get_task` | any | Read task + subtasks + phase outputs (context) |
| `report_progress` | any | Post a progress note |
| `update_task_output` | any | Persist the phase's raw output/artifacts |
| `create_subtasks` | planning | Replace the subtask list (returns created ids) |
| `update_subtask_status` | doing+ | Advance a subtask: `pending`→`in_progress`→`completed` |
| `complete_phase` | any | Advance to the next lane (with optional summary) |
| `reopen_phase` | reviewing | Send back one lane with a reason (e.g. validating→doing) |

---

## Project structure

```
cmd/kanbanai/              # CLI entrypoint (Cobra): serve / migrate / version
config/                    # config loader (Viper)
internal/
  domain/                  # pure domain: entity, event, port, repository, query
  application/             # use cases, phase orchestrator, prompt builder, DTOs
  adapter/
    in/
      http/                # Gin routes, handlers, middleware, SSE, live, frontend mount
      mcp/                 # MCP server + 7 tool definitions/handlers
    out/
      harness/             # adapter, command builder, phase-config provider
      persistence/         # SQLite repos + migrations + task-with-phases query
      livetail/            # in-memory ring buffer + SSE subscribers
    bootstrap/             # wiring + DI container assembly
  di/                      # DI container
scripts/                   # harness wrappers + systemd launcher
  pi-harness.{sh,mjs}      # pi (SDK, REST-shimmed tools)
  claude-harness.{sh,mjs}  # Claude Code (native MCP)
  run-kanbanai.sh          # sources .env.dev (+.env.local), sets PATH, execs server
frontend/                  # React 18 + MUI 5 + Vite
docs/                      # architecture, api-spec, configuration, domain, … (SPEC)
web/                       # built frontend (served by Go at /)
```

---

## Development

```bash
make build            # go build -o bin/kanbanai
make dev              # go run cmd/kanbanai/main.go serve
make migrate          # run DB migrations
make test             # go test ./... -v -race -cover
make vet              # go vet ./...
make lint             # golangci-lint run ./...
make frontend-dev     # vite dev server (HMR, :3000)
make frontend-build   # tsc + vite build → web/
make docker-build     # multi-stage image
make clean            # rm -rf bin/ data/
```

The Go server serves the built frontend from `KANBANAI_WEB_DIR` (`./web`) at `/`.
During frontend development, run `make frontend-dev` (Vite, port 3000) pointed at
the Go API; in production, `make frontend-build` and the Go binary serves it.

Tests: Go unit tests cover the domain, use cases, orchestrator (prior-context
injection, reopen feedback, phase transitions), prompt builder, subtask use
cases, and the phase-config provider. Migration count is asserted in
`sqlite_test.go`.

**Harness smoke tests** (`make smoke-test`) exercise each harness wrapper
(`scripts/pi-harness.mjs`, `scripts/claude-harness.mjs`) against a mock SDK /
mock `claude` binary and an in-process mock KanbanAI REST API — no real model
or running server required. They assert: customTools/MCP registration, live-tail
streaming (text/thinking/tool markers), the happy path (explicit `complete_phase`),
and the safety-net behaviors (auto-save output, auto-complete, tolerant
pre-complete guard, reopen path, prompt-failure). Run individually with
`make smoke-pi` / `make smoke-claude`.

---

## Operations (systemd)

The dev instance runs as a **user** systemd service so it survives logout and
logs to journald. The launcher (`scripts/run-kanbanai.sh`) sources `.env.dev`,
optionally `.env.local` (Claude auth), pins the nvm node bin + `~/.local/bin`
on `PATH`, and sets `PI_PKG_DIR`.

```bash
# run detached (creates/starts kanbanai.service)
systemd-run --user --unit=kanbanai.service \
  /home/breno/projects/kanbanai/scripts/run-kanbanai.sh

# manage
systemctl --user status|stop|restart kanbanai.service
journalctl --user -u kanbanai.service -f        # logs
```

> **Restart caveat**: `livetail.Store` is in-memory. Restarting the service
> wipes live buffers and orphans any in-progress task (the harness is killed but
> the task stays `in_progress` with nothing running). Use **Retry** or
> **Resume** on the card to recover. (Auto-revival on startup is a known TODO.)

---

## Troubleshooting

- **Card stuck `in_progress`, nothing running** — the service was restarted
  mid-dispatch (live store is in-memory). Click **Retry** (or **Pause** then
  **Resume**) on the card.
- **`spawn claude ENOENT` / `pi SDK package not found`** — the harness binary
  isn't on the service `PATH`. `run-kanbanai.sh` adds `~/.local/bin` (claude)
  and the nvm node bin (pi). If you run the binary directly, ensure both are on
  `PATH` and `PI_PKG_DIR` is set.
- **Claude: "Not logged in"** — running headless without keyring access. Put
  `ANTHROPIC_*` env vars in `.env.local` (gitignored) and ensure
  `run-kanbanai.sh` sources it.
- **Model "may not exist or you may not have access"** — the model string isn't
  recognized by the harness's provider. For Claude, use an alias
  (`sonnet`/`opus`/`haiku`) mapped in `ANTHROPIC_DEFAULT_*_MODEL`, or a full id;
  for pi, use the `provider/model` form your provider expects.
- **Agent refuses to finalize / lane "fails"** — weak models sometimes
  misjudge MCP tool results. Use a stronger model on that lane, or rely on the
  safety-net (it auto-completes). Check the live tail for the agent's reasoning.
- **Per-lane change not taking effect** — confirm via `GET /api/v1/config/phases`
  that the override persisted; empty fields inherit the env default, they don't
  "clear" it.
- **Migrations** — run `make migrate`. Count is asserted in tests; if you add a
  migration, update `sqlite_test.go`.

---

KanbanAI is a Go + React project built on hexagonal architecture. See
`docs/` and `SPEC.md` for the full design.