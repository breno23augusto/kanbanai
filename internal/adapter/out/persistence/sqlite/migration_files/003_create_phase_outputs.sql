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

CREATE INDEX IF NOT EXISTS idx_phase_outputs_task_id ON phase_outputs(task_id);
