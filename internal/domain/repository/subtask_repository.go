package repository

import (
	"context"
	"kanbanai/internal/domain/entity"
)

// SubtaskRepository persists the subtasks created during planning and updated
// across the implementation lanes. FindByTask returns subtasks ordered by their
// creation/position so the UI renders them in the order the planner defined.
type SubtaskRepository interface {
	Create(ctx context.Context, subtask *entity.Subtask) error
	Update(ctx context.Context, subtask *entity.Subtask) error
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, id string) (*entity.Subtask, error)
	FindByTask(ctx context.Context, taskID string) ([]*entity.Subtask, error)
	DeleteByTask(ctx context.Context, taskID string) error
}