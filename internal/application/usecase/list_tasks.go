package usecase

import (
	"context"
	"fmt"

	"kanbanai/internal/application/dto"
	"kanbanai/internal/domain/repository"
)

type ListTasks struct {
	taskRepo    repository.TaskRepository
	subtaskRepo repository.SubtaskRepository
}

func NewListTasks(repo repository.TaskRepository, subtaskRepo repository.SubtaskRepository) *ListTasks {
	return &ListTasks{taskRepo: repo, subtaskRepo: subtaskRepo}
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
		out := &dto.TaskOutput{
			ID:           t.ID,
			Title:        t.Title,
			Description:  t.Description,
			CurrentPhase: t.CurrentPhase,
			Status:       t.Status,
			Priority:     t.Priority,
			Version:      t.Version,
			ErrorMessage: t.ErrorMessage,
			Workspace:    t.Workspace,
			ReopenReason: t.ReopenReason,
			CreatedAt:    t.CreatedAt,
			UpdatedAt:    t.UpdatedAt,
		}
		// Load subtasks so the board card can render per-subtask status. The
		// board is small, so a per-task fetch is acceptable and keeps the list
		// response self-contained (no extra round-trips from the frontend).
		if uc.subtaskRepo != nil {
			items, err := uc.subtaskRepo.FindByTask(ctx, t.ID)
			if err != nil {
				return nil, fmt.Errorf("list subtasks for task %s: %w", t.ID, err)
			}
			out.Subtasks = toSubtaskDTOs(items)
			out.SubtaskSummary = dto.SubtaskSummaryFrom(toEntitySubtasks(items))
		}
		result = append(result, out)
	}

	return result, nil
}
