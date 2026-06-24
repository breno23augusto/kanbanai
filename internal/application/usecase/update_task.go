package usecase

import (
	"context"
	"fmt"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/repository"
	"kanbanai/internal/application/dto"
	"time"
)

type UpdateTask struct {
	taskRepo   repository.TaskRepository
	dispatcher event.Dispatcher
}

func NewUpdateTask(repo repository.TaskRepository, disp event.Dispatcher) *UpdateTask {
	return &UpdateTask{taskRepo: repo, dispatcher: disp}
}

func (uc *UpdateTask) Execute(ctx context.Context, id string, input dto.CreateTaskInput, version int) (*dto.TaskOutput, error) {
	task, err := uc.taskRepo.Find(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}

	if task.Version != version {
		return nil, fmt.Errorf("concurrent modification: version mismatch")
	}

	task.Title = input.Title
	task.Description = input.Description
	task.Priority = input.Priority
	task.UpdatedAt = time.Now()

	if err := uc.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	uc.dispatcher.Publish(event.Event{
		Type:    event.TaskUpdated,
		TaskID:  task.ID,
		Payload: task,
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
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}, nil
}
