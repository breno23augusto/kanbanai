package harness

import (
	"context"
	"fmt"
	"sync"

	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/repository"
)

// PhaseConfigProviderMemory is the runtime-mutable source of truth for per-phase
// harness configuration. It merges the env-derived defaults (immutable, built by
// BuildPhaseConfigs) with DB overrides (editable from the UI). An override field
// that is "" / 0 inherits the default — identical to the env-var semantics.
//
// The harness adapter calls Get on every Dispatch, so UI edits take effect on
// the next phase dispatch without a server restart.
type PhaseConfigProviderMemory struct {
	defaults  map[entity.Phase]entity.PhaseConfig // immutable, env-seeded
	overrides map[entity.Phase]entity.PhaseConfig // DB-loaded, "" / 0 = inherit
	repo      repository.PhaseConfigRepository
	mu       sync.RWMutex
}

func NewPhaseConfigProvider(defaults map[entity.Phase]entity.PhaseConfig, repo repository.PhaseConfigRepository) *PhaseConfigProviderMemory {
	return &PhaseConfigProviderMemory{
		defaults:  defaults,
		overrides: make(map[entity.Phase]entity.PhaseConfig),
		repo:      repo,
	}
}

// merge returns the effective config for a phase: the env default with any
// non-empty DB override field applied on top.
func (p *PhaseConfigProviderMemory) merge(phase entity.Phase) entity.PhaseConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()

	base := p.defaults[phase] // every PhaseOrder entry is seeded, so always present
	ov, ok := p.overrides[phase]
	if !ok {
		return base
	}
	merged := base
	if ov.ModelName != "" {
		merged.ModelName = ov.ModelName
	}
	if ov.HarnessCmd != "" {
		merged.HarnessCmd = ov.HarnessCmd
	}
	if ov.MaxRetries > 0 {
		merged.MaxRetries = ov.MaxRetries
	}
	if ov.TimeoutSec > 0 {
		merged.TimeoutSec = ov.TimeoutSec
	}
	return merged
}

func (p *PhaseConfigProviderMemory) Get(phase entity.Phase) entity.PhaseConfig {
	return p.merge(phase)
}

// Snapshot returns the effective config for all non-terminal lanes in PhaseOrder.
func (p *PhaseConfigProviderMemory) Snapshot() []entity.PhaseConfig {
	out := make([]entity.PhaseConfig, 0, len(entity.PhaseOrder))
	for _, phase := range entity.PhaseOrder {
		if phase.IsTerminal() {
			continue
		}
		out = append(out, p.merge(phase))
	}
	return out
}

// Defaults returns the env-derived baseline (without DB overrides), in PhaseOrder.
func (p *PhaseConfigProviderMemory) Defaults() []entity.PhaseConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]entity.PhaseConfig, 0, len(entity.PhaseOrder))
	for _, phase := range entity.PhaseOrder {
		if phase.IsTerminal() {
			continue
		}
		if d, ok := p.defaults[phase]; ok {
			out = append(out, d)
		}
	}
	return out
}

func (p *PhaseConfigProviderMemory) Reload(ctx context.Context) error {
	overrides, err := p.repo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("load phase_config overrides: %w", err)
	}
	p.mu.Lock()
	p.overrides = make(map[entity.Phase]entity.PhaseConfig, len(overrides))
	for _, ov := range overrides {
		p.overrides[ov.Phase] = ov
	}
	p.mu.Unlock()
	return nil
}

// Replace writes the given overrides to the DB and refreshes the in-memory cache.
// Each field "" / 0 means "inherit default"; passing a PhaseConfig whose fields are
// all zero effectively resets that lane to the env default.
func (p *PhaseConfigProviderMemory) Replace(ctx context.Context, overrides []entity.PhaseConfig) error {
	if err := p.repo.UpsertAll(ctx, overrides); err != nil {
		return fmt.Errorf("persist phase_configs: %w", err)
	}
	return p.Reload(ctx)
}