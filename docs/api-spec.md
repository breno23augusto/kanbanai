# KanbanAI — Especificação da API REST & SSE

A API REST do KanbanAI utiliza o prefixo `/api/v1/` e retorna payloads estruturados em JSON. Todos os payloads de sucesso e erro seguem um padrão fixo.

---

## 1. Tabela de Endpoints

| Método   | Rota | Parâmetros | HTTP Status | Descrição |
|----------|------|------------|-------------|-----------|
| `GET`    | `/api/v1/health` | - | `200 OK` | Health check da API e conexão SQLite |
| `POST`   | `/api/v1/tasks` | **Body**: `{ "title": "string", "description": "string", "priority": int }` | `201 Created` | Cria task e inicia fluxo Kanban |
| `GET`    | `/api/v1/tasks` | **Query**: `?phase=string&status=string&limit=10&offset=0` | `200 OK` | Lista tasks com paginação e filtros |
| `GET`    | `/api/v1/tasks/:id` | **Path**: `id` (ULID) | `200 OK` | Detalhes da task com outputs de cada fase |
| `PUT`    | `/api/v1/tasks/:id` | **Path**: `id`, **Body**: `{ "title": "string", "description": "string", "priority": int, "version": int }` | `200 OK` | Atualização com optimistic locking |
| `DELETE` | `/api/v1/tasks/:id` | **Path**: `id` | `204 No Content` | Remove task e mata harnesses ativos |
| `GET`    | `/api/v1/tasks/:id/timeline` | **Path**: `id` | `200 OK` | Timeline de eventos da task |
| `POST`   | `/api/v1/tasks/:id/retry` | **Path**: `id` | `200 OK` | Reinicia fase em estado `failed` |
| `POST`   | `/api/v1/tasks/:id/pause` | **Path**: `id` | `200 OK` | Pausa o harness em execução e marca status `paused` |
| `POST`   | `/api/v1/tasks/:id/resume` | **Path**: `id` | `200 OK` | Retoma a fase atual de uma task `paused` |
| `POST`   | `/api/v1/tasks/:id/complete` | **Path**: `id`, **Body** (opcional): `{ "phase": string, "summary": string }` | `200 OK` | Marca a fase atual como concluída (bridge para harnesses não-MCP) |
| `POST`   | `/api/v1/tasks/:id/reopen` | **Path**: `id`, **Body** (opcional): `{ "target_phase": string, "reason": string }` | `200 OK` | Move a task de volta a uma fase anterior (rework) e a re-despacha. Fallback HTTP do tool MCP `reopen_phase` (SPEC §6.3.7) |
| `GET`    | `/api/v1/events` | - | `200 OK` (Stream) | Endpoint SSE para eventos em tempo real |

---

## 2. Payloads JSON

### 2.1 POST `/api/v1/tasks` — Criar Task

**Request**:
```json
{
  "title": "Configurar SQLite local",
  "description": "Criar migrations e conexao thread-safe",
  "priority": 2
}
```

**Response `201 Created`**:
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

### 2.2 GET `/api/v1/tasks/:id` — Buscar Task com Outputs

**Response `200 OK`**:
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

### 2.3 POST `/api/v1/tasks/:id/reopen` — Reabrir Fase (Rework)

Move a task de volta a uma fase **anterior** e a re-despacha. É o fallback HTTP
do tool MCP `reopen_phase`, para harnesses sem client MCP (ex.: pi). A fase alvo
precisa preceder a fase atual; para re-executar a **mesma** fase use `/retry`.

**Request**:
```json
{ "target_phase": "doing", "reason": "checkWinner retorna undefined em empate; critério X não atendido" }
```
Se `target_phase` for omitido, reabre para a fase imediatamente anterior.

**Response `200 OK`**:
```json
{
  "success": true,
  "data": {
    "task_id": "01J...",
    "target_phase": "doing",
    "status": "reopened",
    "message": "task moved back and target phase dispatched"
  }
}
```

Em caso de violação (fase alvo não-anterior, task inativa) retorna `400`/`409`.
Publica o evento SSE `lane.reopened` (`{from, to, reason}`).

### 2.4 Erro Padronizado — `409 Conflict`

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

## 3. Mecanismo SSE

O endpoint `/api/v1/events` mantém uma conexão HTTP persistente (`Connection: keep-alive`, `Content-Type: text/event-stream`).

### 3.1 Formato dos Eventos

Exemplo de evento de alteração de raia:
```eventstream
event: task.status_changed
data: {"task_id":"01J185V1WXP8B4K67R2C8V7Y8E","title":"Configurar SQLite local","current_phase":"doing","status":"in_progress","version":3}
```

Exemplo de progresso reportado por um agente:
```eventstream
event: phase.doing.progress
data: {"task_id":"01J185V1WXP8B4K67R2C8V7Y8E","phase":"doing","message":"Escrevendo arquivo sqlite_connection.go","progress_percentage":45}
```

### 3.2 Como o Frontend consome

1. **Carregamento Inicial**: `useTasks.ts` executa `GET /api/v1/tasks` e popula as 6 raias do `KanbanBoard`.
2. **Conexão Real-Time**: `useSSE.ts` inicia `new EventSource('/api/v1/events')`.
3. **Mutações Reativas**:
   - `task.status_changed` → move card para nova coluna (`current_phase`)
   - `phase.*.progress` → exibe barra de progresso e log de atividades
   - `task.created` / `task.deleted` → insere/remove card instantaneamente
   - `task.paused` / `task.resumed` → atualiza status do card (pausado/retomado)
   - `lane.reopened` → task voltou a uma fase anterior para rework (atualiza coluna)
   - Em caso de desconexão, EventSource reconecta automaticamente e re-lista tasks

---

## 4. Servindo o Frontend Estático

O servidor Go serve os arquivos estáticos do frontend (build do React) na rota raiz `/`:

```go
// internal/adapter/in/http/router.go
func SetupRoutes(r *gin.Engine, container *di.Container, webDir string) {
    api := r.Group("/api/v1")
    // ... rotas da API ...

    r.Static("/assets", filepath.Join(webDir, "assets"))
    r.StaticFile("/", filepath.Join(webDir, "index.html"))
    r.NoRoute(func(c *gin.Context) {
        c.File(filepath.Join(webDir, "index.html")) // SPA fallback
    })
}
```

O diretório `webDir` é configurável via `KANBANAI_WEB_DIR` (default: `./web`).

---

## 5. Versionamento da API

A API REST segue versionamento explícito `/api/v1/`. Todas as rotas são prefixadas com `/api/v1/`.
