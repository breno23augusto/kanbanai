package service

import (
	"context"
	"fmt"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/port"
	"kanbanai/internal/domain/repository"
	"time"
)

type PhaseOrchestrator struct {
	taskRepo        repository.TaskRepository
	phaseOutputRepo repository.PhaseOutputRepository
	harnessAdapter  port.HarnessPort
	promptBuilder   *PromptBuilder
	dispatcher      event.Dispatcher
}

func NewPhaseOrchestrator(
	taskRepo repository.TaskRepository,
	phaseOutputRepo repository.PhaseOutputRepository,
	harnessAdapter port.HarnessPort,
	promptBuilder *PromptBuilder,
	dispatcher event.Dispatcher,
) *PhaseOrchestrator {
	return &PhaseOrchestrator{
		taskRepo:        taskRepo,
		phaseOutputRepo: phaseOutputRepo,
		harnessAdapter:  harnessAdapter,
		promptBuilder:   promptBuilder,
		dispatcher:      dispatcher,
	}
}

func (o *PhaseOrchestrator) StartFlow(ctx context.Context, task *entity.Task) error {
	o.dispatcher.Publish(event.Event{
		Type:      event.PhasePlanningStarted,
		TaskID:    task.ID,
		Payload:   map[string]any{"phase": entity.PhasePlanning},
		Timestamp: time.Now(),
	})

	prompt, err := o.promptBuilder.Build(string(entity.PhasePlanning), PromptData{
		Title:       task.Title,
		Description: task.Description,
		ID:          task.ID,
		Phase:       string(entity.PhasePlanning),
	})
	if err != nil {
		return fmt.Errorf("prompt build: %w", err)
	}

	if err := o.harnessAdapter.Dispatch(ctx, task, entity.PhasePlanning, prompt); err != nil {
		return fmt.Errorf("harness dispatch: %w", err)
	}

	return nil
}

func (o *PhaseOrchestrator) AdvancePhase(ctx context.Context, taskID string) error {
	task, err := o.taskRepo.Find(ctx, taskID)
	if err != nil {
		return fmt.Errorf("find task: %w", err)
	}

	fromPhase := task.CurrentPhase

	if task.CurrentPhase.IsTerminal() {
		// Already at done; ensure task is completed.
		task.Status = entity.StatusCompleted
		task.UpdatedAt = time.Now()
		if err := o.taskRepo.Update(ctx, task); err != nil {
			return fmt.Errorf("update task: %w", err)
		}
		return nil
	}

	nextPhase, hasNext := fromPhase.Next()
	if !hasNext || nextPhase.IsTerminal() {
		task.CurrentPhase = entity.PhaseDone
		task.Status = entity.StatusCompleted
		task.UpdatedAt = time.Now()
		if err := o.taskRepo.Update(ctx, task); err != nil {
			return fmt.Errorf("update task: %w", err)
		}
		o.dispatcher.Publish(event.Event{
			Type:      event.PhaseDoneReached,
			TaskID:    task.ID,
			Payload:   map[string]any{"phase": entity.PhaseDone},
			Timestamp: time.Now(),
		})
		return nil
	}

	task.CurrentPhase = nextPhase
	task.Status = entity.StatusPending
	task.UpdatedAt = time.Now()
	if err := o.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	o.dispatcher.Publish(event.Event{
		Type:      event.LaneTransitionCompleted,
		TaskID:    task.ID,
		Payload:   map[string]any{"from": fromPhase, "to": nextPhase},
		Timestamp: time.Now(),
	})

	return o.dispatchPhase(ctx, task, nextPhase)
}

func (o *PhaseOrchestrator) dispatchPhase(ctx context.Context, task *entity.Task, phase entity.Phase) error {
	task.Status = entity.StatusInProgress
	task.UpdatedAt = time.Now()
	if err := o.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	o.dispatcher.Publish(event.Event{
		Type:      event.PhaseEvent(string(phase), "started"),
		TaskID:    task.ID,
		Payload:   map[string]any{"phase": phase},
		Timestamp: time.Now(),
	})

	prompt, err := o.promptBuilder.Build(string(phase), PromptData{
		Title:       task.Title,
		Description: task.Description,
		ID:          task.ID,
		Phase:       string(phase),
	})
	if err != nil {
		return fmt.Errorf("prompt build: %w", err)
	}

	return o.harnessAdapter.Dispatch(ctx, task, phase, prompt)
}

func (o *PhaseOrchestrator) KillProcess(taskID string) {
	o.harnessAdapter.KillProcess(taskID)
}

func (o *PhaseOrchestrator) HandleRetry(ctx context.Context, taskID string, phase entity.Phase, attempt int, maxRetries int) {
	if attempt > maxRetries {
		task, err := o.taskRepo.Find(ctx, taskID)
		if err != nil {
			return
		}
		task.Status = entity.StatusFailed
		task.UpdatedAt = time.Now()
		_ = o.taskRepo.Update(ctx, task)
		o.dispatcher.Publish(event.Event{
			Type:      event.PhaseEvent(string(phase), "failed"),
			TaskID:    taskID,
			Payload:   map[string]any{"phase": phase, "attempt": attempt},
			Timestamp: time.Now(),
		})
		return
	}

	time.Sleep(time.Duration(2*attempt) * time.Second)
	o.dispatcher.Publish(event.Event{
		Type:      event.PhaseEvent(string(phase), "retry"),
		TaskID:    taskID,
		Payload:   map[string]any{"phase": phase, "attempt": attempt},
		Timestamp: time.Now(),
	})

	task, err := o.taskRepo.Find(ctx, taskID)
	if err != nil {
		return
	}
	_ = o.dispatchPhase(ctx, task, phase)
}