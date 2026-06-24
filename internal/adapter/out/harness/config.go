package harness

import (
	"kanbanai/internal/domain/entity"
)

func BuildPhaseConfigs(
	defaultCmd string,
	defaultModel string,
	defaultMaxRetries int,
	defaultTimeoutSec int,
	phaseOverrides map[entity.Phase]PhaseHarnessConfig,
) map[entity.Phase]entity.PhaseConfig {
	configs := make(map[entity.Phase]entity.PhaseConfig)

	for _, phase := range entity.PhaseOrder {
		cfg := entity.PhaseConfig{
			Phase:      phase,
			HarnessCmd: defaultCmd,
			ModelName:  defaultModel,
			MaxRetries: defaultMaxRetries,
			TimeoutSec: defaultTimeoutSec,
		}

		if override, ok := phaseOverrides[phase]; ok {
			if override.Cmd != "" {
				cfg.HarnessCmd = override.Cmd
			}
			if override.Model != "" {
				cfg.ModelName = override.Model
			}
			if override.MaxRetries > 0 {
				cfg.MaxRetries = override.MaxRetries
			}
			if override.TimeoutSec > 0 {
				cfg.TimeoutSec = override.TimeoutSec
			}
		}

		configs[phase] = cfg
	}

	return configs
}

type PhaseHarnessConfig struct {
	Cmd        string
	Model      string
	MaxRetries int
	TimeoutSec int
}
