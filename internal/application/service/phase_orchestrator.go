package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
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
	subtaskRepo     repository.SubtaskRepository
	harnessAdapter  port.HarnessPort
	promptBuilder   *PromptBuilder
	dispatcher      event.Dispatcher
	retryAttempts   map[string]int
	mu              sync.Mutex
}

func NewPhaseOrchestrator(
	taskRepo repository.TaskRepository,
	phaseOutputRepo repository.PhaseOutputRepository,
	subtaskRepo repository.SubtaskRepository,
	harnessAdapter port.HarnessPort,
	promptBuilder *PromptBuilder,
	dispatcher event.Dispatcher,
) *PhaseOrchestrator {
	return &PhaseOrchestrator{
		taskRepo:        taskRepo,
		phaseOutputRepo: phaseOutputRepo,
		subtaskRepo:     subtaskRepo,
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
		t.ErrorMessage = "" // fresh run clears any prior failure reason
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
		t.ReopenReason = "" // the rework was accepted forward; clear the feedback
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
		t.ErrorMessage = "" // a (re)dispatch is a fresh attempt; clear stale reason
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

	data := PromptData{
		Title:       updated.Title,
		Description: updated.Description,
		ID:          updated.ID,
		Phase:       string(phase),
		Workspace:   updated.Workspace,
	}

	// Review phases (validating, testing) need the upstream context to judge
	// the implementation against the original prompt. Populate the acceptance
	// criteria (planning + todo outputs) and the implementation report (doing
	// output) so the reviewer can perform the comparison without having to
	// fetch every artifact itself. Errors here are non-fatal: a missing prior
	// output just yields an empty block, and the prompt instructs the harness
	// to fall back to the get_task MCP tool (SPEC §6.3.7).
	if phase == entity.PhaseValidating || phase == entity.PhaseTesting {
		if err := o.populatePriorContext(ctx, updated.ID, phase, &data); err != nil {
			slog.Warn("orchestrator: failed to load prior phase context",
				"taskID", updated.ID, "phase", phase, "error", err)
		}
	}

	// Implementation/review lanes (doing, validating, testing) work from the
	// tracked subtask checklist created in planning. Inject it so the harness
	// knows what to do and can report per-subtask progress. Non-fatal on error.
	if phase == entity.PhaseDoing || phase == entity.PhaseValidating || phase == entity.PhaseTesting {
		if text, err := o.loadSubtasksText(ctx, updated.ID); err != nil {
			slog.Warn("orchestrator: failed to load subtasks",
				"taskID", updated.ID, "phase", phase, "error", err)
		} else {
			data.Subtasks = text
		}
	}

	// Rework lanes (doing, todo, planning) carry the reason a downstream phase
	// sent the task back, plus that downstream review's output, so the next
	// attempt addresses the concrete problems instead of re-running blind.
	// Non-fatal: an empty result just omits the block (first run).
	if phase == entity.PhaseDoing || phase == entity.PhaseTodo || phase == entity.PhasePlanning {
		if text, err := o.loadReviewFeedback(ctx, updated, phase); err != nil {
			slog.Warn("orchestrator: failed to load review feedback",
				"taskID", updated.ID, "phase", phase, "error", err)
		} else {
			data.ReviewFeedback = text
		}
	}

	prompt, err := o.promptBuilder.Build(string(phase), data)
	if err != nil {
		return fmt.Errorf("prompt build: %w", err)
	}

	return o.harnessAdapter.Dispatch(ctx, updated, phase, prompt)
}

// populatePriorContext loads the phase outputs produced upstream of the given
// review phase and fills the AcceptanceCriteria and ImplementationReport fields
// of data. Acceptance criteria come from the refinement lanes that precede the
// implementation (planning + todo); the implementation report comes from the
// doing lane. Each block combines the raw Output (saved via update_task_output)
// and the Summary (saved via complete_phase), preferring the richer raw output
// and falling back to the summary. Empty phases collapse to a single
// "(no output saved)" line so the prompt stays well-formed and the reviewer is
// explicitly told to fetch the data via get_task.
func (o *PhaseOrchestrator) populatePriorContext(ctx context.Context, taskID string, phase entity.Phase, data *PromptData) error {
	outputs, err := o.phaseOutputRepo.FindByFilters(ctx, repository.Criteria{
		{Key: "task_id", Value: taskID, Operator: repository.OpEquals},
	})
	if err != nil {
		return fmt.Errorf("find phase outputs: %w", err)
	}

	byPhase := make(map[entity.Phase]*entity.PhaseOutput, len(outputs))
	for _, po := range outputs {
		// Keep the most recently updated record per phase (there should be only
		// one, but defend against duplicates from retries).
		if cur, ok := byPhase[po.Phase]; ok && !po.UpdatedAt.After(cur.UpdatedAt) {
			continue
		}
		byPhase[po.Phase] = po
	}

	block := func(phases ...entity.Phase) string {
		var parts []string
		for _, p := range phases {
			po, ok := byPhase[p]
			if !ok {
				parts = append(parts, fmt.Sprintf("[%s] (no output saved)", p))
				continue
			}
			text := po.Output
			if strings.TrimSpace(text) == "" {
				text = po.Summary
			}
			if strings.TrimSpace(text) == "" {
				parts = append(parts, fmt.Sprintf("[%s] (no output saved)", p))
				continue
			}
			parts = append(parts, fmt.Sprintf("[%s]\n%s", p, text))
		}
		return strings.Join(parts, "\n\n")
	}

	switch phase {
	case entity.PhaseValidating, entity.PhaseTesting:
		data.AcceptanceCriteria = block(entity.PhasePlanning, entity.PhaseTodo)
		data.ImplementationReport = block(entity.PhaseDoing)
	}
	return nil
}

// loadSubtasksText renders the task's tracked subtasks as an ordered,
// status-annotated checklist suitable for injection into a phase prompt. It is
// used by the doing/validating/testing lanes so the harness has the concrete
// breakdown to work through and report against. Returns a "(no subtasks
// created yet)" placeholder when none exist, prompting the harness to proceed
// directly or fetch via get_task.
func (o *PhaseOrchestrator) loadSubtasksText(ctx context.Context, taskID string) (string, error) {
	items, err := o.subtaskRepo.FindByTask(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("find subtasks: %w", err)
	}
	if len(items) == 0 {
		return "(no subtasks created yet — proceed by implementing the task directly, or call get_task to confirm)", nil
	}
	var b strings.Builder
	for _, st := range items {
		fmt.Fprintf(&b, "- [%s] %s (id: %s)\n", st.Status, st.Title, st.ID)
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

// loadReviewFeedback assembles the rework guidance for a re-dispatched
// implementation/refinement lane (doing/todo/planning): the explicit reason a
// downstream phase cited when it called reopen_phase, plus the most recent
// downstream review output (e.g. the validating FAIL report). This is what
// makes a rework attempt smarter than the first run — the harness sees the
// concrete problems to fix. Returns "" on a first run (no reopen reason and
// no downstream output), in which case the prompt omits the block entirely.
func (o *PhaseOrchestrator) loadReviewFeedback(ctx context.Context, task *entity.Task, phase entity.Phase) (string, error) {
	reason := strings.TrimSpace(task.ReopenReason)

	outputs, err := o.phaseOutputRepo.FindByFilters(ctx, repository.Criteria{
		{Key: "task_id", Value: task.ID, Operator: repository.OpEquals},
	})
	if err != nil {
		return "", fmt.Errorf("find phase outputs: %w", err)
	}

	// Downstream reviews are outputs from phases that come AFTER the current
	// one (e.g. for doing: validating/testing). Keep the most recent per phase.
	var downstream []*entity.PhaseOutput
	seen := make(map[entity.Phase]*entity.PhaseOutput, len(outputs))
	for _, po := range outputs {
		if !po.Phase.After(phase) {
			continue
		}
		if cur, ok := seen[po.Phase]; ok && !po.UpdatedAt.After(cur.UpdatedAt) {
			continue
		}
		seen[po.Phase] = po
	}
	for _, po := range seen {
		downstream = append(downstream, po)
	}
	// most recent first
	sort.Slice(downstream, func(i, j int) bool {
		return downstream[i].UpdatedAt.After(downstream[j].UpdatedAt)
	})

	if reason == "" && len(downstream) == 0 {
		return "", nil
	}

	var b strings.Builder
	if reason != "" {
		b.WriteString("REASON FOR REWORK (cited by the reviewer who sent this task back):\n")
		b.WriteString(reason)
		b.WriteString("\n\n")
	}
	if len(downstream) > 0 {
		b.WriteString("DOWNSTREAM REVIEW (the full review that triggered the rollback — address every item):\n")
		for _, po := range downstream {
			text := po.Output
			if strings.TrimSpace(text) == "" {
				text = po.Summary
			}
			if strings.TrimSpace(text) == "" {
				continue
			}
			fmt.Fprintf(&b, "[%s]\n%s\n\n", po.Phase, text)
		}
	}
	b.WriteString("Address every issue above before calling complete_phase. Do NOT re-run the same implementation that produced these findings.")
	return strings.TrimRight(b.String(), "\n"), nil
}

func (o *PhaseOrchestrator) KillProcess(taskID string) {
	o.harnessAdapter.KillProcess(taskID)
}

// HandleRetry is triggered by the HarnessErrorOccurred subscriber. It tracks
// the attempt count for the current phase and either re-dispatches the phase
// (with linear backoff) or marks the task as failed when retries are exhausted
// (SPEC §6.3.6 / §13.2 / §32.3). The reason carries the captured harness
// output/error so the failure is explainable on the frontend.
func (o *PhaseOrchestrator) HandleRetry(ctx context.Context, taskID string, phase entity.Phase, maxRetries int, reason string) {
	o.mu.Lock()
	o.retryAttempts[taskID]++
	attempt := o.retryAttempts[taskID]
	o.mu.Unlock()

	if attempt > maxRetries {
		if _, err := o.retryUpdate(ctx, taskID, func(t *entity.Task) {
			t.Status = entity.StatusFailed
			t.ErrorMessage = formatFailureReason(phase, attempt, reason)
			t.UpdatedAt = time.Now()
		}); err != nil {
			slog.Error("orchestrator: failed to mark task as failed", "taskID", taskID, "error", err)
		}
		o.dispatcher.Publish(event.Event{
			Type:      event.PhaseEvent(string(phase), "failed"),
			TaskID:    taskID,
			Payload:   map[string]any{"phase": phase, "attempt": attempt, "reason": formatFailureReason(phase, attempt, reason)},
			Timestamp: time.Now(),
		})
		return
	}

	time.Sleep(time.Duration(2*attempt) * time.Second)

	o.dispatcher.Publish(event.Event{
		Type:      event.PhaseEvent(string(phase), "retry"),
		TaskID:    taskID,
		Payload:   map[string]any{"phase": phase, "attempt": attempt, "reason": reason},
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

// PauseTask stops the running harness process for the task and marks the task
// as paused. Only tasks currently in_progress can be paused. The harness
// process is killed via KillProcess, which marks the termination as
// intentional so monitorProcess does not trigger an automatic retry/failure
// (SPEC §32.3). A paused task can later be resumed via ResumeTask or edited
// via the regular UpdateTask use case.
func (o *PhaseOrchestrator) PauseTask(ctx context.Context, taskID string) error {
	task, err := o.taskRepo.Find(ctx, taskID)
	if err != nil {
		return fmt.Errorf("find task: %w", err)
	}
	if task.Status != entity.StatusInProgress {
		return fmt.Errorf("task %s is not running (status=%s), cannot pause", taskID, task.Status)
	}

	// Stop the running harness first so it does not keep producing MCP calls
	// or race with the status update.
	o.harnessAdapter.KillProcess(taskID)

	updated, err := o.retryUpdate(ctx, taskID, func(t *entity.Task) {
		t.Status = entity.StatusPaused
		t.UpdatedAt = time.Now()
	})
	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	o.dispatcher.Publish(event.Event{
		Type:      event.TaskPaused,
		TaskID:    updated.ID,
		Payload:   map[string]any{"phase": updated.CurrentPhase},
		Timestamp: time.Now(),
	})
	return nil
}

// ResumeTask re-dispatches the current phase of a paused task, resetting the
// retry counter. Only tasks in the paused state can be resumed. This mirrors
// RestartPhase but guards the state transition so a non-paused task cannot be
// accidentally (re)dispatched.
func (o *PhaseOrchestrator) ResumeTask(ctx context.Context, taskID string) error {
	task, err := o.taskRepo.Find(ctx, taskID)
	if err != nil {
		return fmt.Errorf("find task: %w", err)
	}
	if task.Status != entity.StatusPaused {
		return fmt.Errorf("task %s is not paused (status=%s), cannot resume", taskID, task.Status)
	}

	o.resetAttempts(taskID)

	if err := o.dispatchPhase(ctx, task, task.CurrentPhase); err != nil {
		return fmt.Errorf("dispatch phase: %w", err)
	}

	o.dispatcher.Publish(event.Event{
		Type:      event.TaskResumed,
		TaskID:    taskID,
		Payload:   map[string]any{"phase": task.CurrentPhase},
		Timestamp: time.Now(),
	})
	return nil
}

// ReopenPhase moves a task BACK to an earlier lane and re-dispatches it,
// so problems detected by a downstream phase (typically Validating) get
// reworked instead of being carried forward (SPEC §6.3.7).
//
// It is exposed to harnesses via the reopen_phase MCP tool and the
// POST /tasks/:id/reopen HTTP endpoint. The calling harness is expected to be
// in a later phase (e.g. validating) and to stop its own work after the call
// returns, exactly like complete_phase; ReopenPhase therefore does NOT kill
// the caller's process — it just moves the lane and dispatches the target
// phase, mirroring how AdvancePhase hands off to the next lane.
//
// Guards:
//   - targetPhase must be a known, non-terminal phase that precedes the
//     task's current phase (target.Before(current)). Reopening to the current
//     or a later phase is rejected; for same-phase re-runs use RestartPhase.
//   - the task must be in an active status (pending/in_progress), so a failed
//     task cannot be silently resurrected from a downstream phase.
func (o *PhaseOrchestrator) ReopenPhase(ctx context.Context, taskID string, targetPhase entity.Phase, reason string) error {
	task, err := o.taskRepo.Find(ctx, taskID)
	if err != nil {
		return fmt.Errorf("find task: %w", err)
	}
	if task.Status != entity.StatusPending && task.Status != entity.StatusInProgress {
		return fmt.Errorf("task %s is not active (status=%s), cannot reopen", taskID, task.Status)
	}
	if targetPhase.IsTerminal() || targetPhase.Index() < 0 {
		return fmt.Errorf("invalid target phase %s for reopen", targetPhase)
	}
	if !targetPhase.Before(task.CurrentPhase) {
		return fmt.Errorf("cannot reopen to %s: it does not precede the current phase %s",
			targetPhase, task.CurrentPhase)
	}

	fromPhase := task.CurrentPhase

	updated, err := o.retryUpdate(ctx, taskID, func(t *entity.Task) {
		t.CurrentPhase = targetPhase
		t.Status = entity.StatusPending
		t.ErrorMessage = ""     // rework is a fresh attempt; clear stale reason
		t.ReopenReason = reason // carry the reviewer's feedback into the next dispatch
		t.UpdatedAt = time.Now()
	})
	if err != nil {
		return fmt.Errorf("update task for reopen: %w", err)
	}
	o.resetAttempts(taskID)

	o.dispatcher.Publish(event.Event{
		Type:      event.LaneReopened,
		TaskID:    updated.ID,
		Payload:   map[string]any{"from": fromPhase, "to": targetPhase, "reason": reason},
		Timestamp: time.Now(),
	})

	return o.dispatchPhase(ctx, updated, targetPhase)
}

// formatFailureReason builds the human-readable explanation stored on the task
// (and carried by the phase.<phase>.failed event) when retries are exhausted.
// It combines the process wait error with the captured harness stdout/stderr,
// which is where the actual cause lives (e.g. "agent prompt failed: ...",
// "complete failed: 404 ...").
func formatFailureReason(phase entity.Phase, attempt int, reason string) string {
	header := fmt.Sprintf("Phase \"%s\" failed after %d attempt(s).", phase, attempt)
	body := strings.TrimSpace(reason)
	if body == "" {
		return header + " No harness output was captured."
	}
	return header + "\n\n" + body
}
