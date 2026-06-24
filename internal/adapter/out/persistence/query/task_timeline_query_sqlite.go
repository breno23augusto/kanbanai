package query

import (
	"database/sql"
	"fmt"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/query"
)

type TaskTimelineQuerySQLite struct {
	db *sql.DB
}

func NewTaskTimelineQuerySQLite(db *sql.DB) *TaskTimelineQuerySQLite {
	return &TaskTimelineQuerySQLite{db: db}
}

func (q *TaskTimelineQuerySQLite) Get(taskID string) (*query.TaskTimelineResult, error) {
	taskQuery := `SELECT id, title, description, current_phase, status, priority, version, error_message, created_at, updated_at
	               FROM tasks WHERE id = ?`
	row := q.db.QueryRow(taskQuery, taskID)

	task := &entity.Task{}
	err := row.Scan(&task.ID, &task.Title, &task.Description, &task.CurrentPhase,
		&task.Status, &task.Priority, &task.Version, &task.ErrorMessage,
		&task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("scan task: %w", err)
	}

	eventsQuery := `SELECT id, task_id, event_type, phase, message, metadata, created_at
	                 FROM task_event_logs WHERE task_id = ? ORDER BY created_at ASC`
	rows, err := q.db.Query(eventsQuery, taskID)
	if err != nil {
		return nil, fmt.Errorf("query event logs: %w", err)
	}
	defer rows.Close()

	var events []entity.TaskEventLog
	for rows.Next() {
		evt := entity.TaskEventLog{}
		var phase, metadataStr string
		if err := rows.Scan(&evt.ID, &evt.TaskID, &evt.EventType, &phase, &evt.Message, &metadataStr, &evt.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event log: %w", err)
		}
		evt.Phase = entity.Phase(phase)
		events = append(events, evt)
	}

	return &query.TaskTimelineResult{
		Task:   *task,
		Events: events,
	}, nil
}
