package dto

type CreateTaskInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	// Workspace is the filesystem path the harness runs in for this task.
	// Empty means use the server's configured default (PI_HARNESS_CWD).
	Workspace    string `json:"workspace"`
}
