package query

import (
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/repository"
)

type TaskWithPhasesResult struct {
	Task         entity.Task
	PhaseOutputs []entity.PhaseOutput
}

type TaskWithPhasesQuery interface {
	Get(taskID string) (*TaskWithPhasesResult, error)
	List(criteria repository.Criteria) ([]*TaskWithPhasesResult, error)
}
