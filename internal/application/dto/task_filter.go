package dto

import "kanbanai/internal/domain/entity"

type TaskFilter struct {
	Phase  *entity.Phase `json:"phase,omitempty"`
	Status *entity.Status `json:"status,omitempty"`
	Limit  int           `json:"limit,omitempty"`
	Offset int           `json:"offset,omitempty"`
}
