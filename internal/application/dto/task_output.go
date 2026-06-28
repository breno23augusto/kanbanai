package dto

import (
	"kanbanai/internal/domain/entity"
	"time"
)

type TaskOutput struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	CurrentPhase entity.Phase  `json:"current_phase"`
	Status       entity.Status `json:"status"`
	Priority     int           `json:"priority"`
	Version      int           `json:"version"`
	ErrorMessage string        `json:"error_message"`
	// Workspace is the harness working directory for this task (empty = server default).
	Workspace   string        `json:"workspace"`
	// Subtasks carries the per-subtask status so the board card can render
	// live progress. Populated by ListTasks and GetTask.
	Subtasks      []SubtaskDTO  `json:"subtasks"`
	// SubtaskSummary is the derived progress aggregate (total/completed/
	// in_progress). Provided for convenience so consumers don't have to
	// recompute it.
	SubtaskSummary SubtaskSummary `json:"subtask_summary"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}
