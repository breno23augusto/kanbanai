package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"kanbanai/internal/application/dto"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/repository"
)

const updateTaskMaxRetries = 3

type UpdateTask struct {
	taskRepo   repository.TaskRepository
	dispatcher event.Dispatcher
}

func NewUpdateTask(repo repository.TaskRepository, disp event.Dispatcher) *UpdateTask {
	return &UpdateTask{taskRepo: repo, dispatcher: disp}
}

func (uc *UpdateTask) Execute(ctx context.Context, id string, input dto.CreateTaskInput, version int) (*dto.TaskOutput, error) {
	for attempt := 0; ; attempt++ {
		task, err := uc.taskRepo.Find(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("find task: %w", err)
		}

		// Client supplied a stale version: surface a concurrent modification
		// error to the caller (HTTP 409) without retrying — the client must
		// reload the data and retry explicitly (SPEC §26).
		if task.Version != version {
			return nil, fmt.Errorf("%w: client version %d does not match current %d",
				repository.ErrConcurrentModification, version, task.Version)
		}

		task.Title = input.Title
		task.Description = input.Description
		task.Priority = input.Priority
		task.Workspace = input.Workspace
		task.UpdatedAt = time.Now()

		if err := uc.taskRepo.Update(ctx, task); err != nil {
			// Race between Find and Update: reload and reapply up to
			// updateTaskMaxRetries times before giving up (SPEC §26).
			if attempt < updateTaskMaxRetries && errors.Is(err, repository.ErrConcurrentModification) {
				continue
			}
			return nil, fmt.Errorf("update task: %w", err)
		}

		uc.dispatcher.Publish(event.Event{
			Type:      event.TaskUpdated,
			TaskID:    task.ID,
			Payload:   task,
			Timestamp: time.Now(),
		})

		return &dto.TaskOutput{
			ID:           task.ID,
			Title:        task.Title,
			Description:  task.Description,
			CurrentPhase: task.CurrentPhase,
			Status:       task.Status,
			Priority:     task.Priority,
			Version:      task.Version,
			ErrorMessage: task.ErrorMessage,
			Workspace:    task.Workspace,
			CreatedAt:    task.CreatedAt,
			UpdatedAt:    task.UpdatedAt,
		}, nil
	}
}