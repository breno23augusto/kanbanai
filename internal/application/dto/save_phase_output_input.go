package dto

import "kanbanai/internal/domain/entity"

type SavePhaseOutputInput struct {
	TaskID  string       `json:"task_id"`
	Phase   entity.Phase `json:"phase"`
	Output  string       `json:"output"`
	Summary string       `json:"summary"`
}
