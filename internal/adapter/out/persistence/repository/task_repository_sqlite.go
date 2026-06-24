package repository

import (
	"context"
	"database/sql"
	"fmt"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/repository"
)

type TaskRepositorySQLite struct {
	db *sql.DB
}

func NewTaskRepositorySQLite(db *sql.DB) *TaskRepositorySQLite {
	return &TaskRepositorySQLite{db: db}
}

func (r *TaskRepositorySQLite) Create(ctx context.Context, task *entity.Task) error {
	query := `INSERT INTO tasks (id, title, description, current_phase, status, priority, version, error_message, created_at, updated_at)
	           VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		task.ID, task.Title, task.Description, string(task.CurrentPhase),
		string(task.Status), task.Priority, task.Version, task.ErrorMessage,
		task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

func (r *TaskRepositorySQLite) Update(ctx context.Context, task *entity.Task) error {
	query := `UPDATE tasks
	           SET title = ?, description = ?, current_phase = ?, status = ?, priority = ?,
	               error_message = ?, version = version + 1, updated_at = ?
	           WHERE id = ? AND version = ?`

	result, err := r.db.ExecContext(ctx, query,
		task.Title, task.Description, string(task.CurrentPhase),
		string(task.Status), task.Priority, task.ErrorMessage, task.UpdatedAt,
		task.ID, task.Version,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("%w for task %s", repository.ErrConcurrentModification, task.ID)
	}

	task.Version++
	return nil
}

func (r *TaskRepositorySQLite) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}

func (r *TaskRepositorySQLite) Find(ctx context.Context, id string) (*entity.Task, error) {
	query := `SELECT id, title, description, current_phase, status, priority, version, error_message, created_at, updated_at
	           FROM tasks WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	task := &entity.Task{}
	err := row.Scan(&task.ID, &task.Title, &task.Description, &task.CurrentPhase,
		&task.Status, &task.Priority, &task.Version, &task.ErrorMessage,
		&task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		return nil, fmt.Errorf("scan task: %w", err)
	}

	return task, nil
}

func (r *TaskRepositorySQLite) FindByFilters(ctx context.Context, criteria repository.Criteria) ([]*entity.Task, error) {
	query := "SELECT id, title, description, current_phase, status, priority, version, error_message, created_at, updated_at FROM tasks WHERE 1=1"
	args := make([]any, 0)

	for _, c := range criteria {
		query += fmt.Sprintf(" AND %s %s ?", c.Key, string(c.Operator))
		args = append(args, c.Value)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*entity.Task
	for rows.Next() {
		task := &entity.Task{}
		if err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.CurrentPhase,
			&task.Status, &task.Priority, &task.Version, &task.ErrorMessage,
			&task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}