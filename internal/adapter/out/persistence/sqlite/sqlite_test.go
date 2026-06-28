package sqlite

import (
	"database/sql"
	"testing"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := dir + "/test.db"
	db, err := NewConnection(dbPath)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := RunMigrations(db, "migration_files"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestMigrationsCreateAllTables(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tables := []string{"tasks", "task_event_logs", "phase_outputs", "schema_migrations"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Fatalf("query %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("table %s not found (count=%d)", table, count)
		}
	}
}

func TestMigrationsIdempotent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Running migrations again should not error and should not duplicate.
	if err := RunMigrations(db, "migration_files"); err != nil {
		t.Fatalf("second migration run failed: %v", err)
	}

	var count int
	err := db.QueryRow("SELECT count(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count != 7 {
		t.Errorf("expected 7 recorded migrations, got %d", count)
	}
}