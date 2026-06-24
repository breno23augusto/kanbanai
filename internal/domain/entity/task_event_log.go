package entity

import "time"

type TaskEventLog struct {
	ID        string
	TaskID    string
	EventType string
	Phase     Phase
	Message   string
	Metadata  map[string]any
	CreatedAt time.Time
}
