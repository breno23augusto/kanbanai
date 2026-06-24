package repository

import (
	"context"
	"kanbanai/internal/domain/entity"
)

type PhaseOutputRepository interface {
	Create(ctx context.Context, output *entity.PhaseOutput) error
	Update(ctx context.Context, output *entity.PhaseOutput) error
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, id string) (*entity.PhaseOutput, error)
	FindByFilters(ctx context.Context, criteria Criteria) ([]*entity.PhaseOutput, error)
}
