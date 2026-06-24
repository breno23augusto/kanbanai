package port

import (
	"context"
	"kanbanai/internal/domain/entity"
)

type HarnessPort interface {
	Dispatch(ctx context.Context, task *entity.Task, phase entity.Phase, prompt string) error
	KillProcess(taskID string)
}
