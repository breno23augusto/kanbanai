package repository

import (
	"context"
	"kanbanai/internal/domain/entity"
)

type TaskEventLogRepository interface {
	Create(ctx context.Context, log *entity.TaskEventLog) error
	Update(ctx context.Context, log *entity.TaskEventLog) error
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, id string) (*entity.TaskEventLog, error)
	FindByFilters(ctx context.Context, criteria Criteria) ([]*entity.TaskEventLog, error)
}
