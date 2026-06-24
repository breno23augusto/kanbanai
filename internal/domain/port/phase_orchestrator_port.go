package port

import (
	"context"
	"kanbanai/internal/domain/entity"
)

type PhaseOrchestratorPort interface {
	StartFlow(ctx context.Context, task *entity.Task) error
	AdvancePhase(ctx context.Context, taskID string) error
	RestartPhase(ctx context.Context, taskID string) error
}