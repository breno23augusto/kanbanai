# KanbanAI — Schema do Banco (SQLite)

## 1. Tabela `tasks`

```sql
-- migration_files/001_create_tasks.sql
CREATE TABLE IF NOT EXISTS tasks (
    id            TEXT PRIMARY KEY,
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    current_phase TEXT NOT NULL DEFAULT 'planning',
    status        TEXT NOT NULL DEFAULT 'pending',
    priority      INTEGER NOT NULL DEFAULT 0,
    version       INTEGER NOT NULL DEFAULT 1, -- Optimistic locking
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tasks_current_phase ON tasks(current_phase);
CREATE INDEX idx_tasks_status ON tasks(status);
```

## 2. Tabela `task_event_logs`

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

## 3. Tabela `phase_outputs`

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

## 4. Controle de Concorrência (Optimistic Locking)

A coluna `version` na tabela `tasks` é usada para optimistic locking:

```sql
UPDATE tasks 
SET title = ?, description = ?, current_phase = ?, status = ?, version = version + 1, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND version = ?;
```

Se `rows affected = 0`, o repositório retorna `ErrConcurrentModification`. Os Use Cases capturam este erro e efetuam retries automáticos (até 3 tentativas).
