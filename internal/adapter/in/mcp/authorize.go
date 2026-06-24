package mcp

import (
	"context"
	"fmt"
	"os"

	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/repository"
	"kanbanai/internal/di"
)

// validateTaskID enforces the SPEC §32.1 / rules.md §5 authorization: when the
// MCP server process carries KANBANAI_TASK_ID (stdio per-task transport), the
// requested task_id must match it exactly. This check applies to every tool,
// including read-only ones.
func validateTaskID(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("task_id is required")
	}
	if envTaskID := os.Getenv("KANBANAI_TASK_ID"); envTaskID != "" && envTaskID != taskID {
		return fmt.Errorf("task_id %s does not match authorized task %s", taskID, envTaskID)
	}
	return nil
}

// authorize additionally validates the task's live state for write tools: the
// task must exist, its current_phase must match the requested phase, and it
// must be in an active (pending/in_progress) status. This prevents cross-phase
// or post-completion writes under the shared SSE transport, where the server
// process does not carry KANBANAI_TASK_ID.
func authorize(ctx context.Context, container *di.Container, taskID, phase string) error {
	if err := validateTaskID(taskID); err != nil {
		return err
	}

	taskRepo, ok := container.Resolve("taskRepo").(repository.TaskRepository)
	if !ok {
		return fmt.Errorf("task repository not available")
	}

	task, err := taskRepo.Find(ctx, taskID)
	if err != nil {
		return fmt.Errorf("authorize: find task: %w", err)
	}

	if phase != "" && task.CurrentPhase != entity.Phase(phase) {
		return fmt.Errorf("phase %s is not the current phase of task %s (current: %s)",
			phase, taskID, task.CurrentPhase)
	}

	if task.Status != entity.StatusPending && task.Status != entity.StatusInProgress {
		return fmt.Errorf("task %s is not active (status=%s)", taskID, task.Status)
	}

	return nil
}