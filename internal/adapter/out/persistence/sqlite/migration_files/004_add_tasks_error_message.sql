-- 004_add_tasks_error_message.sql
-- Persist the reason a task entered the failed state so the frontend can
-- surface it (SPEC §16.1 / §32.3). Cleared on retry/resume/advance.
ALTER TABLE tasks ADD COLUMN error_message TEXT NOT NULL DEFAULT '';