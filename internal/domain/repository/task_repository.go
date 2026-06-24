package repository

import (
	"context"
	"kanbanai/internal/domain/entity"
)

type TaskRepository interface {
	Create(ctx context.Context, task *entity.Task) error
	Update(ctx context.Context, task *entity.Task) error
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, id string) (*entity.Task, error)
	FindByFilters(ctx context.Context, criteria Criteria) ([]*entity.Task, error)
}
