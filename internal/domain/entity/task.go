package entity

import "time"

type Task struct {
	ID           string
	Title        string
	Description  string
	CurrentPhase Phase
	Status       Status
	Priority     int
	Version      int
	ErrorMessage string
	// Workspace is the filesystem path the harness runs in for this task
	// (its cwd). Empty means use the server's configured default workspace.
	Workspace string
	// ReopenReason holds the reason a downstream phase (e.g. validating) sent the
	// task back for rework via reopen_phase. It is set on ReopenPhase and cleared
	// when the reworked lane advances forward again, so the next attempt's
	// prompt carries the precise, actionable feedback that caused the rollback.
	ReopenReason string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
