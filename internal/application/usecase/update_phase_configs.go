package usecase

import (
	"context"
	"fmt"
	"time"

	"kanbanai/internal/application/dto"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/port"
)

// UpdatePhaseConfigs replaces all per-lane overrides from the UI and fires a
// PhaseConfigsUpdated event so the frontend refreshes. Each input lane's empty
// fields mean "inherit env default".
type UpdatePhaseConfigs struct {
	provider  port.PhaseConfigProvider
	dispatcher event.Dispatcher
}

func NewUpdatePhaseConfigs(provider port.PhaseConfigProvider, dispatcher event.Dispatcher) *UpdatePhaseConfigs {
	return &UpdatePhaseConfigs{provider: provider, dispatcher: dispatcher}
}

func (uc *UpdatePhaseConfigs) Execute(ctx context.Context, inputs []dto.PhaseConfigInput) ([]dto.PhaseConfigDTO, error) {
	// Validate: only non-terminal lanes, each recognized.
	allowed := make(map[entity.Phase]struct{}, len(entity.PhaseOrder))
	for _, ph := range entity.PhaseOrder {
		if !ph.IsTerminal() {
			allowed[ph] = struct{}{}
		}
	}

	overrides := make([]entity.PhaseConfig, 0, len(inputs))
	seen := make(map[entity.Phase]struct{}, len(inputs))
	for _, in := range inputs {
		phase := entity.Phase(in.Phase)
		if _, ok := allowed[phase]; !ok {
			return nil, fmt.Errorf("invalid or terminal phase: %q", in.Phase)
		}
		if _, dup := seen[phase]; dup {
			return nil, fmt.Errorf("duplicate phase in request: %q", in.Phase)
		}
		if in.MaxRetries < 0 {
			return nil, fmt.Errorf("max_retries must be >= 0 for phase %q", in.Phase)
		}
		if in.TimeoutSec < 0 {
			return nil, fmt.Errorf("timeout_sec must be >= 0 for phase %q", in.Phase)
		}
		seen[phase] = struct{}{}
		overrides = append(overrides, entity.PhaseConfig{
			Phase:      phase,
			ModelName:  in.Model,
			HarnessCmd: in.HarnessCmd,
			MaxRetries: in.MaxRetries,
			TimeoutSec: in.TimeoutSec,
		})
	}

	if err := uc.provider.Replace(ctx, overrides); err != nil {
		return nil, fmt.Errorf("update phase configs: %w", err)
	}

	now := time.Now()
	uc.dispatcher.Publish(event.Event{
		Type:      event.PhaseConfigsUpdated,
		Payload:   map[string]any{"phases": len(overrides)},
		Timestamp: now,
	})

	// Return the freshly-merged effective view (same shape as GET).
	effectives := uc.provider.Snapshot()
	defaults := uc.provider.Defaults()
	defMap := make(map[entity.Phase]entity.PhaseConfig, len(defaults))
	for _, d := range defaults {
		defMap[d.Phase] = d
	}
	out := make([]dto.PhaseConfigDTO, 0, len(effectives))
	for _, eff := range effectives {
		out = append(out, dto.NewPhaseConfigDTO(eff, defMap[eff.Phase]))
	}
	return out, nil
}