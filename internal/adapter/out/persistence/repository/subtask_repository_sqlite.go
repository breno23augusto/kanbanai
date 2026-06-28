package repository

import (
	"context"
	"database/sql"
	"fmt"

	"kanbanai/internal/domain/entity"
)

type SubtaskRepositorySQLite struct {
	db *sql.DB
}

func NewSubtaskRepositorySQLite(db *sql.DB) *SubtaskRepositorySQLite {
	return &SubtaskRepositorySQLite{db: db}
}

func (r *SubtaskRepositorySQLite) Create(ctx context.Context, st *entity.Subtask) error {
	query := `INSERT INTO subtasks (id, task_id, title, status, sort_order, created_at, updated_at)
	           VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		st.ID, st.TaskID, st.Title, string(st.Status), st.Order, st.CreatedAt, st.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert subtask: %w", err)
	}
	return nil
}

func (r *SubtaskRepositorySQLite) Update(ctx context.Context, st *entity.Subtask) error {
	query := `UPDATE subtasks SET title = ?, status = ?, sort_order = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, st.Title, string(st.Status), st.Order, st.UpdatedAt, st.ID)
	if err != nil {
		return fmt.Errorf("update subtask: %w", err)
	}
	return nil
}

func (r *SubtaskRepositorySQLite) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM subtasks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete subtask: %w", err)
	}
	return nil
}

func (r *SubtaskRepositorySQLite) DeleteByTask(ctx context.Context, taskID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM subtasks WHERE task_id = ?", taskID)
	if err != nil {
		return fmt.Errorf("delete subtasks by task: %w", err)
	}
	return nil
}

func (r *SubtaskRepositorySQLite) Find(ctx context.Context, id string) (*entity.Subtask, error) {
	query := `SELECT id, task_id, title, status, sort_order, created_at, updated_at FROM subtasks WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	st := &entity.Subtask{}
	var status string
	if err := row.Scan(&st.ID, &st.TaskID, &st.Title, &status, &st.Order, &st.CreatedAt, &st.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("subtask not found: %s", id)
		}
		return nil, fmt.Errorf("scan subtask: %w", err)
	}
	st.Status = entity.SubtaskStatus(status)
	return st, nil
}

func (r *SubtaskRepositorySQLite) FindByTask(ctx context.Context, taskID string) ([]*entity.Subtask, error) {
	query := `SELECT id, task_id, title, status, sort_order, created_at, updated_at
	            FROM subtasks WHERE task_id = ? ORDER BY sort_order ASC, created_at ASC`
	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("query subtasks: %w", err)
	}
	defer rows.Close()

	var items []*entity.Subtask
	for rows.Next() {
		st := &entity.Subtask{}
		var status string
		if err := rows.Scan(&st.ID, &st.TaskID, &st.Title, &status, &st.Order, &st.CreatedAt, &st.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan subtask: %w", err)
		}
		st.Status = entity.SubtaskStatus(status)
		items = append(items, st)
	}
	return items, nil
}