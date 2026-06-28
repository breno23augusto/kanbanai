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
	Workspace    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
