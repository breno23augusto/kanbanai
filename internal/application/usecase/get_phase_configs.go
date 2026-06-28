package usecase

import (
	"context"

	"kanbanai/internal/application/dto"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/port"
)

// GetPhaseConfigs returns the effective per-lane config (merged) plus the env
// defaults, for the GET /config/phases endpoint.
type GetPhaseConfigs struct {
	provider port.PhaseConfigProvider
}

func NewGetPhaseConfigs(provider port.PhaseConfigProvider) *GetPhaseConfigs {
	return &GetPhaseConfigs{provider: provider}
}

func (uc *GetPhaseConfigs) Execute(ctx context.Context) ([]dto.PhaseConfigDTO, error) {
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