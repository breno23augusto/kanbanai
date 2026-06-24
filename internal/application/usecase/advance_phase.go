package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/repository"
	"kanbanai/pkg/uid"
)

const advancePhaseMaxRetries = 3

type AdvancePhase struct {
	taskRepo        repository.TaskRepository
	phaseOutputRepo repository.PhaseOutputRepository
	dispatcher      event.Dispatcher
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

// Execute is invoked by the complete_phase MCP tool. It persists the phase
// completion: saves the summary as a PhaseOutput (preserving any output
// previously saved via update_task_output), marks the task as completed and
// publishes the phase.<phase>.completed event. It deliberately does NOT start
// the next phase: the PhaseOrchestrator reacts to the published event and
// advances the lane (SPEC §6.3.5 / rules.md §3).
func (uc *AdvancePhase) Execute(ctx context.Context, taskID string, phase entity.Phase, summary string) error {
	if err := uc.savePhaseSummary(ctx, taskID, phase, summary); err != nil {
		return fmt.Errorf("save phase output: %w", err)
	}

	if err := uc.markTaskCompleted(ctx, taskID); err != nil {
		return err
	}

	uc.dispatcher.Publish(event.Event{
		Type:      event.PhaseEvent(string(phase), "completed"),
		TaskID:    taskID,
		Payload:   map[string]any{"phase": phase, "summary": summary},
		Timestamp: time.Now(),
	})

	return nil
}

// savePhaseSummary upserts the phase summary, preserving the raw output that
// the harness may have saved earlier through update_task_output.
func (uc *AdvancePhase) savePhaseSummary(ctx context.Context, taskID string, phase entity.Phase, summary string) error {
	existing, err := uc.phaseOutputRepo.FindByFilters(ctx, repository.Criteria{
		{Key: "task_id", Value: taskID, Operator: repository.OpEquals},
		{Key: "phase", Value: string(phase), Operator: repository.OpEquals},
	})
	if err != nil {
		return fmt.Errorf("find phase output: %w", err)
	}

	now := time.Now()
	if len(existing) > 0 {
		po := existing[0]
		po.Summary = summary
		po.UpdatedAt = now
		if err := uc.phaseOutputRepo.Update(ctx, po); err != nil {
			return fmt.Errorf("update phase output: %w", err)
		}
		return nil
	}

	po := &entity.PhaseOutput{
		ID:        uid.New(),
		TaskID:    taskID,
		Phase:     phase,
		Summary:   summary,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.phaseOutputRepo.Create(ctx, po); err != nil {
		return fmt.Errorf("create phase output: %w", err)
	}
	return nil
}

// markTaskCompleted sets status = completed, retrying on optimistic locking
// conflicts (SPEC §26 / rules.md §1).
func (uc *AdvancePhase) markTaskCompleted(ctx context.Context, taskID string) error {
	for attempt := 0; ; attempt++ {
		task, err := uc.taskRepo.Find(ctx, taskID)
		if err != nil {
			return fmt.Errorf("find task: %w", err)
		}
		task.Status = entity.StatusCompleted
		task.UpdatedAt = time.Now()
		if err := uc.taskRepo.Update(ctx, task); err != nil {
			if attempt < advancePhaseMaxRetries && errors.Is(err, repository.ErrConcurrentModification) {
				continue
			}
			return fmt.Errorf("update task status: %w", err)
		}
		return nil
	}
}