package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/port"
	"kanbanai/internal/domain/repository"
)

const orchestratorUpdateMaxRetries = 3

type PhaseOrchestrator struct {
	taskRepo        repository.TaskRepository
	phaseOutputRepo repository.PhaseOutputRepository
	harnessAdapter  port.HarnessPort
	promptBuilder   *PromptBuilder
	dispatcher      event.Dispatcher
	retryAttempts   map[string]int
	mu              sync.Mutex
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
		retryAttempts:   make(map[string]int),
	}
}

// retryUpdate reloads the task, applies the mutation and persists it, retrying
// on optimistic locking conflicts (SPEC §26). It returns the persisted task.
func (o *PhaseOrchestrator) retryUpdate(ctx context.Context, taskID string, apply func(*entity.Task)) (*entity.Task, error) {
	for attempt := 0; ; attempt++ {
		task, err := o.taskRepo.Find(ctx, taskID)
		if err != nil {
			return nil, fmt.Errorf("find task: %w", err)
		}
		apply(task)
		if err := o.taskRepo.Update(ctx, task); err != nil {
			if attempt < orchestratorUpdateMaxRetries && errors.Is(err, repository.ErrConcurrentModification) {
				continue
			}
			return nil, fmt.Errorf("update task: %w", err)
		}
		return task, nil
	}
}

func (o *PhaseOrchestrator) resetAttempts(taskID string) {
	o.mu.Lock()
	o.retryAttempts[taskID] = 0
	o.mu.Unlock()
}

func (o *PhaseOrchestrator) StartFlow(ctx context.Context, task *entity.Task) error {
	o.resetAttempts(task.ID)

	// Move the task to in_progress as the harness for the planning phase starts
	// (SPEC §32.2).
	updated, err := o.retryUpdate(ctx, task.ID, func(t *entity.Task) {
		t.Status = entity.StatusInProgress
		t.UpdatedAt = time.Now()
	})
	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	o.dispatcher.Publish(event.Event{
		Type:      event.PhasePlanningStarted,
		TaskID:    updated.ID,
		Payload:   map[string]any{"phase": entity.PhasePlanning},
		Timestamp: time.Now(),
	})

	prompt, err := o.promptBuilder.Build(string(entity.PhasePlanning), PromptData{
		Title:       updated.Title,
		Description: updated.Description,
		ID:          updated.ID,
		Phase:       string(entity.PhasePlanning),
	})
	if err != nil {
		return fmt.Errorf("prompt build: %w", err)
	}

	if err := o.harnessAdapter.Dispatch(ctx, updated, entity.PhasePlanning, prompt); err != nil {
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

	if fromPhase.IsTerminal() {
		_, err := o.retryUpdate(ctx, taskID, func(t *entity.Task) {
			t.Status = entity.StatusCompleted
			t.UpdatedAt = time.Now()
		})
		if err != nil {
			return fmt.Errorf("update task: %w", err)
		}
		return nil
	}

	nextPhase, hasNext := fromPhase.Next()
	if !hasNext || nextPhase.IsTerminal() {
		updated, err := o.retryUpdate(ctx, taskID, func(t *entity.Task) {
			t.CurrentPhase = entity.PhaseDone
			t.Status = entity.StatusCompleted
			t.UpdatedAt = time.Now()
		})
		if err != nil {
			return fmt.Errorf("update task: %w", err)
		}
		o.dispatcher.Publish(event.Event{
			Type:      event.PhaseDoneReached,
			TaskID:    updated.ID,
			Payload:   map[string]any{"phase": entity.PhaseDone},
			Timestamp: time.Now(),
		})
		return nil
	}

	updated, err := o.retryUpdate(ctx, taskID, func(t *entity.Task) {
		t.CurrentPhase = nextPhase
		t.Status = entity.StatusPending
		t.UpdatedAt = time.Now()
	})
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	o.resetAttempts(taskID)

	o.dispatcher.Publish(event.Event{
		Type:      event.LaneTransitionCompleted,
		TaskID:    updated.ID,
		Payload:   map[string]any{"from": fromPhase, "to": nextPhase},
		Timestamp: time.Now(),
	})

	return o.dispatchPhase(ctx, updated, nextPhase)
}

func (o *PhaseOrchestrator) dispatchPhase(ctx context.Context, task *entity.Task, phase entity.Phase) error {
	updated, err := o.retryUpdate(ctx, task.ID, func(t *entity.Task) {
		t.Status = entity.StatusInProgress
		t.UpdatedAt = time.Now()
	})
	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	o.dispatcher.Publish(event.Event{
		Type:      event.PhaseEvent(string(phase), "started"),
		TaskID:    updated.ID,
		Payload:   map[string]any{"phase": phase},
		Timestamp: time.Now(),
	})

	prompt, err := o.promptBuilder.Build(string(phase), PromptData{
		Title:       updated.Title,
		Description: updated.Description,
		ID:          updated.ID,
		Phase:       string(phase),
	})
	if err != nil {
		return fmt.Errorf("prompt build: %w", err)
	}

	return o.harnessAdapter.Dispatch(ctx, updated, phase, prompt)
}

func (o *PhaseOrchestrator) KillProcess(taskID string) {
	o.harnessAdapter.KillProcess(taskID)
}

// HandleRetry is triggered by the HarnessErrorOccurred subscriber. It tracks
// the attempt count for the current phase and either re-dispatches the phase
// (with linear backoff) or marks the task as failed when retries are exhausted
// (SPEC §6.3.6 / §13.2 / §32.3).
func (o *PhaseOrchestrator) HandleRetry(ctx context.Context, taskID string, phase entity.Phase, maxRetries int) {
	o.mu.Lock()
	o.retryAttempts[taskID]++
	attempt := o.retryAttempts[taskID]
	o.mu.Unlock()

	if attempt > maxRetries {
		if _, err := o.retryUpdate(ctx, taskID, func(t *entity.Task) {
			t.Status = entity.StatusFailed
			t.UpdatedAt = time.Now()
		}); err != nil {
			slog.Error("orchestrator: failed to mark task as failed", "taskID", taskID, "error", err)
		}
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
		slog.Error("orchestrator: failed to load task for retry", "taskID", taskID, "error", err)
		return
	}
	_ = o.dispatchPhase(ctx, task, phase)
}

// RestartPhase re-dispatches the current phase of a task, resetting the retry
// counter. It is invoked by the manual retry endpoint for tasks stuck in a
// failed state (SPEC §16.1 / §32.3).
func (o *PhaseOrchestrator) RestartPhase(ctx context.Context, taskID string) error {
	task, err := o.taskRepo.Find(ctx, taskID)
	if err != nil {
		return fmt.Errorf("find task: %w", err)
	}
	o.resetAttempts(taskID)
	return o.dispatchPhase(ctx, task, task.CurrentPhase)
}