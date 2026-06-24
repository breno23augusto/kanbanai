package usecase

import (
	"context"
	"fmt"
	"kanbanai/internal/domain/repository"
	"kanbanai/internal/application/dto"
)

type ListTasks struct {
	taskRepo repository.TaskRepository
}

func NewListTasks(repo repository.TaskRepository) *ListTasks {
	return &ListTasks{taskRepo: repo}
}

func (uc *ListTasks) Execute(ctx context.Context, filter dto.TaskFilter) ([]*dto.TaskOutput, error) {
	var criteria repository.Criteria

	if filter.Phase != nil {
		criteria = append(criteria, repository.Criterion{
			Key:      "current_phase",
			Value:    string(*filter.Phase),
			Operator: repository.OpEquals,
		})
	}
	if filter.Status != nil {
		criteria = append(criteria, repository.Criterion{
			Key:      "status",
			Value:    string(*filter.Status),
			Operator: repository.OpEquals,
		})
	}

	tasks, err := uc.taskRepo.FindByFilters(ctx, criteria)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	if filter.Limit > 0 && filter.Offset < len(tasks) {
		end := filter.Offset + filter.Limit
		if end > len(tasks) {
			end = len(tasks)
		}
		tasks = tasks[filter.Offset:end]
	}

	var result []*dto.TaskOutput
	for _, t := range tasks {
		result = append(result, &dto.TaskOutput{
			ID:           t.ID,
			Title:        t.Title,
			Description:  t.Description,
			CurrentPhase: t.CurrentPhase,
			Status:       t.Status,
			Priority:     t.Priority,
			Version:      t.Version,
			ErrorMessage: t.ErrorMessage,
			CreatedAt:    t.CreatedAt,
			UpdatedAt:    t.UpdatedAt,
		})
	}

	return result, nil
}
