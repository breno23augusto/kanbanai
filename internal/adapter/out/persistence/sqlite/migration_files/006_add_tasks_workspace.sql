-- 006_add_tasks_workspace.sql
-- Per-task harness working directory (cwd). When non-empty, the harness process
-- for this task runs in this path instead of the server's configured default
-- (PI_HARNESS_CWD). Lets each task target its own repository/workspace.
ALTER TABLE tasks ADD COLUMN workspace TEXT NOT NULL DEFAULT '';