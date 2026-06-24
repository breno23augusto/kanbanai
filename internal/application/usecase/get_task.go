package usecase

import (
	"context"
	"fmt"
	"kanbanai/internal/domain/query"
	"kanbanai/internal/application/dto"
)

type GetTask struct {
	taskWithPhasesQuery query.TaskWithPhasesQuery
}

func NewGetTask(taskWithPhasesQuery query.TaskWithPhasesQuery) *GetTask {
	return &GetTask{taskWithPhasesQuery: taskWithPhasesQuery}
}

func (uc *GetTask) Execute(ctx context.Context, id string) (*query.TaskWithPhasesResult, error) {
	result, err := uc.taskWithPhasesQuery.Get(id)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	return result, nil
}

// TaskOutputWithPhases is used by the HTTP handler to build the response
type TaskOutputWithPhases struct {
	Task         dto.TaskOutput        `json:"task"`
	PhaseOutputs []dto.PhaseOutputDTO `json:"phase_outputs"`
}
