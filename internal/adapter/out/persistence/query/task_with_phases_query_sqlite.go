package query

import (
	"context"
	"database/sql"
	"fmt"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/query"
	"kanbanai/internal/domain/repository"
)

type TaskWithPhasesQuerySQLite struct {
	db          *sql.DB
	subtaskRepo repository.SubtaskRepository
}

func NewTaskWithPhasesQuerySQLite(db *sql.DB, subtaskRepo repository.SubtaskRepository) *TaskWithPhasesQuerySQLite {
	return &TaskWithPhasesQuerySQLite{db: db, subtaskRepo: subtaskRepo}
}

func (q *TaskWithPhasesQuerySQLite) loadSubtasks(ctx context.Context, taskID string) ([]entity.Subtask, error) {
	if q.subtaskRepo == nil {
		return nil, nil
	}
	items, err := q.subtaskRepo.FindByTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("query subtasks: %w", err)
	}
	out := make([]entity.Subtask, 0, len(items))
	for _, st := range items {
		out = append(out, *st)
	}
	return out, nil
}

func (q *TaskWithPhasesQuerySQLite) Get(taskID string) (*query.TaskWithPhasesResult, error) {
	taskQuery := `SELECT id, title, description, current_phase, status, priority, version, error_message, workspace, reopen_reason, created_at, updated_at
	               FROM tasks WHERE id = ?`
	row := q.db.QueryRow(taskQuery, taskID)

	task := &entity.Task{}
	err := row.Scan(&task.ID, &task.Title, &task.Description, &task.CurrentPhase,
		&task.Status, &task.Priority, &task.Version, &task.ErrorMessage, &task.Workspace, &task.ReopenReason,
		&task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("scan task: %w", err)
	}

	outputsQuery := `SELECT id, task_id, phase, output, summary, created_at, updated_at
	                  FROM phase_outputs WHERE task_id = ? ORDER BY created_at ASC`
	rows, err := q.db.Query(outputsQuery, taskID)
	if err != nil {
		return nil, fmt.Errorf("query phase outputs: %w", err)
	}
	defer rows.Close()

	var outputs []entity.PhaseOutput
	for rows.Next() {
		po := entity.PhaseOutput{}
		var phase string
		if err := rows.Scan(&po.ID, &po.TaskID, &phase, &po.Output, &po.Summary, &po.CreatedAt, &po.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan phase output: %w", err)
		}
		po.Phase = entity.Phase(phase)
		outputs = append(outputs, po)
	}

	subtasks, err := q.loadSubtasks(context.Background(), taskID)
	if err != nil {
		return nil, err
	}

	return &query.TaskWithPhasesResult{
		Task:         *task,
		PhaseOutputs: outputs,
		Subtasks:     subtasks,
	}, nil
}

func (q *TaskWithPhasesQuerySQLite) List(criteria repository.Criteria) ([]*query.TaskWithPhasesResult, error) {
	sqlQuery := "SELECT id, title, description, current_phase, status, priority, version, error_message, workspace, reopen_reason, created_at, updated_at FROM tasks WHERE 1=1"
	args := make([]any, 0)

	for _, c := range criteria {
		sqlQuery += fmt.Sprintf(" AND %s %s ?", c.Key, string(c.Operator))
		args = append(args, c.Value)
	}

	sqlQuery += " ORDER BY created_at DESC"

	rows, err := q.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	var results []*query.TaskWithPhasesResult
	for rows.Next() {
		task := &entity.Task{}
		if err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.CurrentPhase,
			&task.Status, &task.Priority, &task.Version, &task.ErrorMessage, &task.Workspace, &task.ReopenReason,
			&task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}

		outputsQuery := "SELECT id, task_id, phase, output, summary, created_at, updated_at FROM phase_outputs WHERE task_id = ? ORDER BY created_at ASC"
		oRows, err := q.db.Query(outputsQuery, task.ID)
		if err != nil {
			return nil, fmt.Errorf("query phase outputs: %w", err)
		}

		var outputs []entity.PhaseOutput
		for oRows.Next() {
			po := entity.PhaseOutput{}
			var phase string
			if err := oRows.Scan(&po.ID, &po.TaskID, &phase, &po.Output, &po.Summary, &po.CreatedAt, &po.UpdatedAt); err != nil {
				oRows.Close()
				return nil, fmt.Errorf("scan phase output: %w", err)
			}
			po.Phase = entity.Phase(phase)
			outputs = append(outputs, po)
		}
		oRows.Close()

		subtasks, err := q.loadSubtasks(context.Background(), task.ID)
		if err != nil {
			return nil, err
		}

		results = append(results, &query.TaskWithPhasesResult{
			Task:         *task,
			PhaseOutputs: outputs,
			Subtasks:     subtasks,
		})
	}

	return results, nil
}
