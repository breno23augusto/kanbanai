package dto

import (
	"kanbanai/internal/domain/entity"
	"time"
)

// SubtaskDTO is the API/MCP representation of a subtask.
type SubtaskDTO struct {
	ID        string              `json:"id"`
	TaskID    string              `json:"task_id"`
	Title     string              `json:"title"`
	Status    entity.SubtaskStatus `json:"status"`
	Order     int                 `json:"order"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

// SubtaskInput is one item of a create_subtasks batch.
type SubtaskInput struct {
	Title string `json:"title"`
}

// SubtaskSummary is the lightweight progress aggregate attached to a task in
// list responses so the board card can render subtask progress without fetching
// every subtask.
type SubtaskSummary struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	InProgress int `json:"in_progress"`
}

func SubtaskSummaryFrom(items []entity.Subtask) SubtaskSummary {
	s := SubtaskSummary{Total: len(items)}
	for _, st := range items {
		switch st.Status {
		case entity.SubtaskCompleted:
			s.Completed++
		case entity.SubtaskInProgress:
			s.InProgress++
		}
	}
	return s
}