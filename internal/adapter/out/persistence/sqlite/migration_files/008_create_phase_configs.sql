-- Per-phase harness overrides editable from the UI. Each row is an override for
-- one lane (phase). Empty/zero values mean "inherit the env default", mirroring
-- the KANBANAI_HARNESS_<PHASE>_* env vars. The harness adapter merges these over
-- the env defaults at dispatch time (see port.PhaseConfigProvider).
CREATE TABLE IF NOT EXISTS phase_configs (
    phase        TEXT PRIMARY KEY,
    model        TEXT NOT NULL DEFAULT '',
    harness_cmd TEXT NOT NULL DEFAULT '',
    max_retries  INTEGER NOT NULL DEFAULT 0,
    timeout_sec  INTEGER NOT NULL DEFAULT 0,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);