package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/repository"
)

type TaskEventLogRepositorySQLite struct {
	db *sql.DB
}

func NewTaskEventLogRepositorySQLite(db *sql.DB) *TaskEventLogRepositorySQLite {
	return &TaskEventLogRepositorySQLite{db: db}
}

func (r *TaskEventLogRepositorySQLite) Create(ctx context.Context, log *entity.TaskEventLog) error {
	metadataJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `INSERT INTO task_event_logs (id, task_id, event_type, phase, message, metadata, created_at)
	           VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err = r.db.ExecContext(ctx, query,
		log.ID, log.TaskID, log.EventType, string(log.Phase),
		log.Message, string(metadataJSON), log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert event log: %w", err)
	}
	return nil
}

func (r *TaskEventLogRepositorySQLite) Update(ctx context.Context, log *entity.TaskEventLog) error {
	metadataJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `UPDATE task_event_logs SET event_type = ?, phase = ?, message = ?, metadata = ? WHERE id = ?`
	_, err = r.db.ExecContext(ctx, query, log.EventType, string(log.Phase), log.Message, string(metadataJSON), log.ID)
	if err != nil {
		return fmt.Errorf("update event log: %w", err)
	}
	return nil
}

func (r *TaskEventLogRepositorySQLite) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM task_event_logs WHERE id = ?", id)
	return err
}

func (r *TaskEventLogRepositorySQLite) Find(ctx context.Context, id string) (*entity.TaskEventLog, error) {
	query := `SELECT id, task_id, event_type, phase, message, metadata, created_at FROM task_event_logs WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	log := &entity.TaskEventLog{}
	var phase, metadataStr string
	err := row.Scan(&log.ID, &log.TaskID, &log.EventType, &phase, &log.Message, &metadataStr, &log.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("event log not found: %s", id)
		}
		return nil, fmt.Errorf("scan event log: %w", err)
	}

	log.Phase = entity.Phase(phase)
	if metadataStr != "" {
		json.Unmarshal([]byte(metadataStr), &log.Metadata)
	}

	return log, nil
}

func (r *TaskEventLogRepositorySQLite) FindByFilters(ctx context.Context, criteria repository.Criteria) ([]*entity.TaskEventLog, error) {
	query := "SELECT id, task_id, event_type, phase, message, metadata, created_at FROM task_event_logs WHERE 1=1"
	args := make([]any, 0)

	for _, c := range criteria {
		query += fmt.Sprintf(" AND %s %s ?", c.Key, string(c.Operator))
		args = append(args, c.Value)
	}

	query += " ORDER BY created_at ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query event logs: %w", err)
	}
	defer rows.Close()

	var logs []*entity.TaskEventLog
	for rows.Next() {
		log := &entity.TaskEventLog{}
		var phase, metadataStr string
		if err := rows.Scan(&log.ID, &log.TaskID, &log.EventType, &phase, &log.Message, &metadataStr, &log.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event log: %w", err)
		}
		log.Phase = entity.Phase(phase)
		if metadataStr != "" {
			json.Unmarshal([]byte(metadataStr), &log.Metadata)
		}
		logs = append(logs, log)
	}

	return logs, nil
}
