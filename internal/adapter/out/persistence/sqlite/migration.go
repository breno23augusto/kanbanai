package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations applies all pending SQL migration files from migrationsDir,
// in filename order. Each migration is tracked in a schema_migrations table
// so it is applied at most once. Statements are split on a semicolon followed
// by a newline to execute them individually (SQLite's Exec does not support
// multiple statements reliably in all driver versions).
func RunMigrations(db *sql.DB, migrationsDir string) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var sqlFiles []string
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".sql" {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, f := range sqlFiles {
		applied, err := isMigrationApplied(db, f)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}

		if err := execMigration(db, f, string(content)); err != nil {
			return fmt.Errorf("execute migration %s: %w", f, err)
		}

		if _, err := db.Exec("INSERT INTO schema_migrations (filename) VALUES (?)", f); err != nil {
			return fmt.Errorf("record migration %s: %w", f, err)
		}
	}

	return nil
}

func isMigrationApplied(db *sql.DB, filename string) (bool, error) {
	var name string
	err := db.QueryRow("SELECT filename FROM schema_migrations WHERE filename = ?", filename).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check migration %s: %w", filename, err)
	}
	return true, nil
}

func execMigration(db *sql.DB, filename, content string) error {
	// Split into individual statements on lines ending with ';'.
	// SQLite cannot always Exec multi-statement strings via go-sqlite3.
	statements := splitStatements(content)
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" {
			continue
		}
		if _, err := tx.Exec(trimmed); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("exec statement in %s: %w", filename, err)
		}
	}
	return tx.Commit()
}

func splitStatements(content string) []string {
	// Simple splitter: break on ';' at end of line. Good enough for our DDL.
	var statements []string
	var current strings.Builder
	for _, line := range strings.Split(content, "\n") {
		current.WriteString(line)
		current.WriteString("\n")
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, ";") {
			statements = append(statements, current.String())
			current.Reset()
		}
	}
	if strings.TrimSpace(current.String()) != "" {
		statements = append(statements, current.String())
	}
	return statements
}