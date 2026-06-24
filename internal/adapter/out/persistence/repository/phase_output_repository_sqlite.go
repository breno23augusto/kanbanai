package repository

import (
	"context"
	"database/sql"
	"fmt"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/repository"
)

type PhaseOutputRepositorySQLite struct {
	db *sql.DB
}

func NewPhaseOutputRepositorySQLite(db *sql.DB) *PhaseOutputRepositorySQLite {
	return &PhaseOutputRepositorySQLite{db: db}
}

func (r *PhaseOutputRepositorySQLite) Create(ctx context.Context, output *entity.PhaseOutput) error {
	query := `INSERT OR REPLACE INTO phase_outputs (id, task_id, phase, output, summary, created_at, updated_at)
	           VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		output.ID, output.TaskID, string(output.Phase),
		output.Output, output.Summary, output.CreatedAt, output.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert phase output: %w", err)
	}
	return nil
}

func (r *PhaseOutputRepositorySQLite) Update(ctx context.Context, output *entity.PhaseOutput) error {
	query := `UPDATE phase_outputs SET output = ?, summary = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, output.Output, output.Summary, output.UpdatedAt, output.ID)
	if err != nil {
		return fmt.Errorf("update phase output: %w", err)
	}
	return nil
}

func (r *PhaseOutputRepositorySQLite) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM phase_outputs WHERE id = ?", id)
	return err
}

func (r *PhaseOutputRepositorySQLite) Find(ctx context.Context, id string) (*entity.PhaseOutput, error) {
	query := `SELECT id, task_id, phase, output, summary, created_at, updated_at FROM phase_outputs WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	output := &entity.PhaseOutput{}
	var phase string
	err := row.Scan(&output.ID, &output.TaskID, &phase, &output.Output, &output.Summary, &output.CreatedAt, &output.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("phase output not found: %s", id)
		}
		return nil, fmt.Errorf("scan phase output: %w", err)
	}
	output.Phase = entity.Phase(phase)
	return output, nil
}

func (r *PhaseOutputRepositorySQLite) FindByFilters(ctx context.Context, criteria repository.Criteria) ([]*entity.PhaseOutput, error) {
	query := "SELECT id, task_id, phase, output, summary, created_at, updated_at FROM phase_outputs WHERE 1=1"
	args := make([]any, 0)

	for _, c := range criteria {
		query += fmt.Sprintf(" AND %s %s ?", c.Key, string(c.Operator))
		args = append(args, c.Value)
	}

	query += " ORDER BY created_at ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query phase outputs: %w", err)
	}
	defer rows.Close()

	var outputs []*entity.PhaseOutput
	for rows.Next() {
		output := &entity.PhaseOutput{}
		var phase string
		if err := rows.Scan(&output.ID, &output.TaskID, &phase, &output.Output, &output.Summary, &output.CreatedAt, &output.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan phase output: %w", err)
		}
		output.Phase = entity.Phase(phase)
		outputs = append(outputs, output)
	}

	return outputs, nil
}
