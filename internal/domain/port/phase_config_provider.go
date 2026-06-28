package port

import (
	"context"
	"kanbanai/internal/domain/entity"
)

// PhaseConfigProvider is the mutable, runtime source of truth for per-phase
// harness configuration that the harness adapter consults on every Dispatch.
// It merges the env-derived defaults (immutable) with DB overrides (editable
// from the UI): an empty override field inherits the default, exactly like the
// KANBANAI_HARNESS_<PHASE>_* env vars.
type PhaseConfigProvider interface {
	// Get returns the effective config for a phase. Every phase in entity.PhaseOrder
	// is always present (seeded from defaults), so it never returns missing.
	Get(phase entity.Phase) entity.PhaseConfig
	// Snapshot returns the effective config for all non-terminal phases, in
	// PhaseOrder. Used by the GET endpoint and SSE broadcasts.
	Snapshot() []entity.PhaseConfig
	// Defaults returns the env-derived baseline (without DB overrides) so the UI
	// can show "(default: …)" placeholders.
	Defaults() []entity.PhaseConfig
	// Reload re-reads DB overrides and rebuilds the in-memory merged cache.
	Reload(ctx context.Context) error
	// Replace writes the given overrides (one per phase; "" / 0 = inherit default)
	// to the DB and refreshes the cache atomically.
	Replace(ctx context.Context, overrides []entity.PhaseConfig) error
}