package usecase

import (
	"context"
	"fmt"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/repository"
	"time"
)

type DeleteTask struct {
	taskRepo   repository.TaskRepository
	dispatcher event.Dispatcher
}

func NewDeleteTask(repo repository.TaskRepository, disp event.Dispatcher) *DeleteTask {
	return &DeleteTask{taskRepo: repo, dispatcher: disp}
}

func (uc *DeleteTask) Execute(ctx context.Context, id string) error {
	if err := uc.taskRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	uc.dispatcher.Publish(event.Event{
		Type:    event.TaskDeleted,
		TaskID:  id,
		Payload: map[string]string{"task_id": id},
		Timestamp: time.Now(),
	})

	return nil
}
