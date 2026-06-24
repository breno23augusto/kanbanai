package query

import (
	"kanbanai/internal/domain/entity"
)

type TaskTimelineResult struct {
	Task   entity.Task
	Events []entity.TaskEventLog
}

type TaskTimelineQuery interface {
	Get(taskID string) (*TaskTimelineResult, error)
}
