package port

import (
	"context"
	"kanbanai/internal/domain/entity"
)

type PhaseOrchestratorPort interface {
	StartFlow(ctx context.Context, task *entity.Task) error
	AdvancePhase(ctx context.Context, taskID string) error
	RestartPhase(ctx context.Context, taskID string) error
	ReopenPhase(ctx context.Context, taskID string, targetPhase entity.Phase, reason string) error
	PauseTask(ctx context.Context, taskID string) error
	ResumeTask(ctx context.Context, taskID string) error
}