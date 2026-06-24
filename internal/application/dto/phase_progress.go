package dto

import "kanbanai/internal/domain/entity"

type PhaseProgress struct {
	TaskID  string       `json:"task_id"`
	Phase   entity.Phase `json:"phase"`
	Message string       `json:"message"`
}
