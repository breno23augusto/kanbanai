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
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
