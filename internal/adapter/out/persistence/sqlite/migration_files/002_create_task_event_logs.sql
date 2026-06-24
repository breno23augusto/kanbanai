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

CREATE INDEX IF NOT EXISTS idx_task_event_logs_task_id ON task_event_logs(task_id);
CREATE INDEX IF NOT EXISTS idx_task_event_logs_event_type ON task_event_logs(event_type);
