package harness

import (
	"context"
	"testing"

	"kanbanai/internal/domain/entity"
)

// fakePhaseConfigRepo is a minimal in-memory PhaseConfigRepository for provider tests.
type fakePhaseConfigRepo struct {
	overrides []entity.PhaseConfig
}

func (f *fakePhaseConfigRepo) GetAll(ctx context.Context) ([]entity.PhaseConfig, error) {
	return append([]entity.PhaseConfig(nil), f.overrides...), nil
}

func (f *fakePhaseConfigRepo) UpsertAll(ctx context.Context, configs []entity.PhaseConfig) error {
	f.overrides = append([]entity.PhaseConfig(nil), configs...)
	return nil
}

func defaultsWith(planningModel, doingModel string) map[entity.Phase]entity.PhaseConfig {
	return map[entity.Phase]entity.PhaseConfig{
		entity.PhasePlanning:   {Phase: entity.PhasePlanning, ModelName: planningModel, HarnessCmd: "pi-harness.sh", MaxRetries: 1, TimeoutSec: 240},
		entity.PhaseTodo:       {Phase: entity.PhaseTodo, ModelName: "default-model", HarnessCmd: "pi-harness.sh", MaxRetries: 1, TimeoutSec: 240},
		entity.PhaseDoing:     {Phase: entity.PhaseDoing, ModelName: doingModel, HarnessCmd: "pi-harness.sh", MaxRetries: 1, TimeoutSec: 240},
		entity.PhaseValidating: {Phase: entity.PhaseValidating, ModelName: "default-model", HarnessCmd: "pi-harness.sh", MaxRetries: 1, TimeoutSec: 240},
		entity.PhaseTesting:    {Phase: entity.PhaseTesting, ModelName: "default-model", HarnessCmd: "pi-harness.sh", MaxRetries: 1, TimeoutSec: 240},
	}
}

func TestProviderGetInheritsDefaultsWhenNoOverride(t *testing.T) {
	p := NewPhaseConfigProvider(defaultsWith("base", "base"), &fakePhaseConfigRepo{})
	if err := p.Reload(context.Background()); err != nil {
		t.Fatalf("reload: %v", err)
	}
	got := p.Get(entity.PhasePlanning)
	if got.ModelName != "base" {
		t.Errorf("Planning model = %q, want %q (default)", got.ModelName, "base")
	}
}

func TestProviderReplaceAppliesOverridesAndInheritsEmpties(t *testing.T) {
	repo := &fakePhaseConfigRepo{}
	p := NewPhaseConfigProvider(defaultsWith("base", "base"), repo)

	// Override only Doing model + timeout; leave others empty (= inherit).
	err := p.Replace(context.Background(), []entity.PhaseConfig{
		{Phase: entity.PhaseDoing, ModelName: "big-model", TimeoutSec: 900},
	})
	if err != nil {
		t.Fatalf("replace: %v", err)
	}

	doing := p.Get(entity.PhaseDoing)
	if doing.ModelName != "big-model" {
		t.Errorf("Doing model = %q, want %q (override)", doing.ModelName, "big-model")
	}
	if doing.TimeoutSec != 900 {
		t.Errorf("Doing timeout = %d, want 900 (override)", doing.TimeoutSec)
	}
	// Inherited fields keep the default.
	if doing.HarnessCmd != "pi-harness.sh" {
		t.Errorf("Doing cmd = %q, want default %q (inherited)", doing.HarnessCmd, "pi-harness.sh")
	}
	if doing.MaxRetries != 1 {
		t.Errorf("Doing retries = %d, want default 1 (inherited)", doing.MaxRetries)
	}

	// A lane with no override at all returns the pure default.
	plan := p.Get(entity.PhasePlanning)
	if plan.ModelName != "base" {
		t.Errorf("Planning model = %q, want %q (default, untouched)", plan.ModelName, "base")
	}
}

func TestProviderResetToDefault(t *testing.T) {
	repo := &fakePhaseConfigRepo{}
	p := NewPhaseConfigProvider(defaultsWith("base", "base"), repo)

	// Set an override, then clear it (all-zero = inherit default).
	if err := p.Replace(context.Background(), []entity.PhaseConfig{{Phase: entity.PhaseDoing, ModelName: "temp"}}); err != nil {
		t.Fatalf("replace set: %v", err)
	}
	if got := p.Get(entity.PhaseDoing).ModelName; got != "temp" {
		t.Fatalf("expected override applied, got %q", got)
	}

	// Reset by writing all-zero.
	if err := p.Replace(context.Background(), []entity.PhaseConfig{{Phase: entity.PhaseDoing}}); err != nil {
		t.Fatalf("replace reset: %v", err)
	}
	if got := p.Get(entity.PhaseDoing).ModelName; got != "base" {
		t.Errorf("after reset, Doing model = %q, want default %q", got, "base")
	}
}

func TestProviderSnapshotExcludesTerminalAndPreservesOrder(t *testing.T) {
	p := NewPhaseConfigProvider(defaultsWith("base", "base"), &fakePhaseConfigRepo{})
	snap := p.Snapshot()

	want := []entity.Phase{entity.PhasePlanning, entity.PhaseTodo, entity.PhaseDoing, entity.PhaseValidating, entity.PhaseTesting}
	if len(snap) != len(want) {
		t.Fatalf("snapshot has %d phases, want %d", len(snap), len(want))
	}
	for i, ph := range want {
		if snap[i].Phase != ph {
			t.Errorf("snapshot[%d].Phase = %q, want %q", i, snap[i].Phase, ph)
		}
	}
}