package usecase

import (
	"context"
	"fmt"
	"kanbanai/internal/application/dto"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/repository"
	"kanbanai/pkg/uid"
	"time"
)

type CreateTask struct {
	taskRepo   repository.TaskRepository
	dispatcher event.Dispatcher
}

func NewCreateTask(repo repository.TaskRepository, disp event.Dispatcher) *CreateTask {
	return &CreateTask{taskRepo: repo, dispatcher: disp}
}

func (uc *CreateTask) Execute(ctx context.Context, input dto.CreateTaskInput) (*dto.TaskOutput, error) {
	if input.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	now := time.Now()
	task := &entity.Task{
		ID:           uid.New(),
		Title:        input.Title,
		Description:  input.Description,
		CurrentPhase: entity.PhasePlanning,
		Status:       entity.StatusPending,
		Priority:     input.Priority,
		Version:      1,
		Workspace:    input.Workspace,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := uc.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	uc.dispatcher.Publish(event.Event{
		Type:      event.TaskCreated,
		TaskID:    task.ID,
		Payload:   task,
		Timestamp: now,
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
		ReopenReason: task.ReopenReason,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}, nil
}
