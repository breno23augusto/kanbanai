package dto

import (
	"kanbanai/internal/domain/entity"
	"time"
)

type PhaseOutputDTO struct {
	ID        string        `json:"id"`
	TaskID    string        `json:"task_id"`
	Phase     entity.Phase  `json:"phase"`
	Output    string        `json:"output"`
	Summary   string        `json:"summary"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}
