package usecase

import (
	"context"
	"fmt"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/repository"
	"time"
)

type AdvancePhase struct {
	taskRepo         repository.TaskRepository
	phaseOutputRepo  repository.PhaseOutputRepository
	dispatcher       event.Dispatcher
}

func NewAdvancePhase(
	taskRepo repository.TaskRepository,
	phaseOutputRepo repository.PhaseOutputRepository,
	dispatcher event.Dispatcher,
) *AdvancePhase {
	return &AdvancePhase{
		taskRepo:        taskRepo,
		phaseOutputRepo: phaseOutputRepo,
		dispatcher:      dispatcher,
	}
}

func (uc *AdvancePhase) Execute(ctx context.Context, taskID string, phase entity.Phase, summary string) error {
	task, err := uc.taskRepo.Find(ctx, taskID)
	if err != nil {
		return fmt.Errorf("find task: %w", err)
	}

	task.Status = entity.StatusInProgress
	task.UpdatedAt = time.Now()

	if err := uc.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	uc.dispatcher.Publish(event.Event{
		Type:    event.PhaseEvent(string(phase), "completed"),
		TaskID:  taskID,
		Payload: map[string]any{"phase": phase, "summary": summary},
		Timestamp: time.Now(),
	})

	return nil
}
