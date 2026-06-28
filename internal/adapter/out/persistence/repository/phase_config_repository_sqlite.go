package repository

import (
	"context"
	"database/sql"
	"fmt"

	"kanbanai/internal/domain/entity"
)

type PhaseConfigRepositorySQLite struct {
	db *sql.DB
}

func NewPhaseConfigRepositorySQLite(db *sql.DB) *PhaseConfigRepositorySQLite {
	return &PhaseConfigRepositorySQLite{db: db}
}

func (r *PhaseConfigRepositorySQLite) GetAll(ctx context.Context) ([]entity.PhaseConfig, error) {
	query := `SELECT phase, model, harness_cmd, max_retries, timeout_sec
	            FROM phase_configs ORDER BY phase ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query phase_configs: %w", err)
	}
	defer rows.Close()

	var items []entity.PhaseConfig
	for rows.Next() {
		var pc entity.PhaseConfig
		if err := rows.Scan(&pc.Phase, &pc.ModelName, &pc.HarnessCmd, &pc.MaxRetries, &pc.TimeoutSec); err != nil {
			return nil, fmt.Errorf("scan phase_config: %w", err)
		}
		items = append(items, pc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate phase_configs: %w", err)
	}
	return items, nil
}

// UpsertAll replaces the full set of overrides in a single transaction so a UI
// "save" is atomic across all lanes.
func (r *PhaseConfigRepositorySQLite) UpsertAll(ctx context.Context, configs []entity.PhaseConfig) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO phase_configs (phase, model, harness_cmd, max_retries, timeout_sec, updated_at)
	                                      VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	                                      ON CONFLICT(phase) DO UPDATE SET
	                                        model = excluded.model,
	                                        harness_cmd = excluded.harness_cmd,
	                                        max_retries = excluded.max_retries,
	                                        timeout_sec = excluded.timeout_sec,
	                                        updated_at = CURRENT_TIMESTAMP`)
	if err != nil {
		return fmt.Errorf("prepare upsert phase_config: %w", err)
	}
	defer stmt.Close()

	for _, pc := range configs {
		if _, err := stmt.ExecContext(ctx, string(pc.Phase), pc.ModelName, pc.HarnessCmd, pc.MaxRetries, pc.TimeoutSec); err != nil {
			return fmt.Errorf("upsert phase_config %s: %w", pc.Phase, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit phase_configs: %w", err)
	}
	return nil
}