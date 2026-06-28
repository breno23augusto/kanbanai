package repository

import (
	"context"
	"kanbanai/internal/domain/entity"
)

// PhaseConfigRepository persists per-phase harness overrides (model, cmd,
// retries, timeout) that operators edit from the UI. A stored value of "" / 0
// means "inherit the env default" (mirrors the env-var override semantics in
// harness.BuildPhaseConfigs).
type PhaseConfigRepository interface {
	GetAll(ctx context.Context) ([]entity.PhaseConfig, error)
	UpsertAll(ctx context.Context, configs []entity.PhaseConfig) error
}