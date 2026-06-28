package entity

import "time"

// SubtaskStatus is the lifecycle state of a single subtask within a task. The
// planning phase creates subtasks (pending); the doing/validating/testing
// harnesses advance them to in_progress and completed as each sub-activity is
// finished, reporting progress through the update_subtask_status MCP tool.
type SubtaskStatus string

const (
	SubtaskPending    SubtaskStatus = "pending"
	SubtaskInProgress SubtaskStatus = "in_progress"
	SubtaskCompleted  SubtaskStatus = "completed"
)

// IsActive reports whether the status represents ongoing work (used by the
// frontend to render live subtask indicators).
func (s SubtaskStatus) IsActive() bool {
	return s == SubtaskInProgress
}

// IsFinished reports whether the subtask is done.
func (s SubtaskStatus) IsFinished() bool {
	return s == SubtaskCompleted
}

// Subtask is a discrete unit of work identified during the planning phase and
// tracked across the implementation lanes. Each subtask carries its own status
// so the card and detail drawer can show per-subtask progress independently of
// the phase output markdown (SPEC §6.3.x).
type Subtask struct {
	ID        string
	TaskID    string
	Title     string
	Status    SubtaskStatus
	Order     int
	CreatedAt time.Time
	UpdatedAt time.Time
}