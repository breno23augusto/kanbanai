-- 007_add_tasks_reopen_reason.sql
-- Persist the reason a downstream phase (e.g. validating) sent the task back
-- for rework via reopen_phase, so the re-dispatched (earlier) lane's prompt
-- carries the actionable feedback instead of re-running blind.
ALTER TABLE tasks ADD COLUMN reopen_reason TEXT NOT NULL DEFAULT '';