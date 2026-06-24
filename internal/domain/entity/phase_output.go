package entity

import "time"

type PhaseOutput struct {
	ID        string
	TaskID    string
	Phase     Phase
	Output    string
	Summary   string
	CreatedAt time.Time
	UpdatedAt time.Time
}
