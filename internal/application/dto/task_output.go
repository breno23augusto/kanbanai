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
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}
