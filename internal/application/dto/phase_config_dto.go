package dto

import "kanbanai/internal/domain/entity"

// PhaseConfigDTO is the effective per-lane config returned to the UI: the merged
// value (env default + DB override) plus the env default, so the frontend can show
// "(default: …)" placeholders and know whether a field is inherited or overridden.
type PhaseConfigDTO struct {
	Phase      string `json:"phase"`
	Model      string `json:"model"`
	HarnessCmd string `json:"harness_cmd"`
	MaxRetries int    `json:"max_retries"`
	TimeoutSec int    `json:"timeout_sec"`

	// Defaults are the env-derived baseline values (what an empty override falls
	// back to). Read-only on the client.
	DefaultModel      string `json:"default_model"`
	DefaultHarnessCmd string `json:"default_harness_cmd"`
	DefaultMaxRetries int    `json:"default_max_retries"`
	DefaultTimeoutSec int    `json:"default_timeout_sec"`
}

// PhaseConfigInput is one lane's override in a PUT body. Empty/zero = inherit the
// env default (i.e. reset this lane).
type PhaseConfigInput struct {
	Phase      string `json:"phase"`
	Model      string `json:"model"`
	HarnessCmd string `json:"harness_cmd"`
	MaxRetries int    `json:"max_retries"`
	TimeoutSec int    `json:"timeout_sec"`
}

func NewPhaseConfigDTO(eff, def entity.PhaseConfig) PhaseConfigDTO {
	return PhaseConfigDTO{
		Phase:             string(eff.Phase),
		Model:             eff.ModelName,
		HarnessCmd:         eff.HarnessCmd,
		MaxRetries:        eff.MaxRetries,
		TimeoutSec:        eff.TimeoutSec,
		DefaultModel:      def.ModelName,
		DefaultHarnessCmd: def.HarnessCmd,
		DefaultMaxRetries: def.MaxRetries,
		DefaultTimeoutSec: def.TimeoutSec,
	}
}