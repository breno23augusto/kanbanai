package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"kanbanai/internal/adapter/out/persistence/sqlite"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/repository"
)

func setupRepoTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sqlite.NewConnection(dbPath)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	migrationsDir := "../../persistence/sqlite/migration_files"
	if err := sqlite.RunMigrations(db, migrationsDir); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestTaskRepositoryCRUD(t *testing.T) {
	db := setupRepoTestDB(t)
	defer db.Close()

	repo := NewTaskRepositorySQLite(db)
	ctx := context.Background()

	// Create
	task := &entity.Task{
		ID:           "task-1",
		Title:        "Test Task",
		Description:  "A test",
		CurrentPhase: entity.PhasePlanning,
		Status:       entity.StatusPending,
		Priority:     1,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := repo.Create(ctx, task); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Find
	found, err := repo.Find(ctx, "task-1")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found.Title != "Test Task" {
		t.Errorf("title = %s, want Test Task", found.Title)
	}
	if found.CurrentPhase != entity.PhasePlanning {
		t.Errorf("phase = %s, want planning", found.CurrentPhase)
	}

	// Update (optimistic locking)
	found.Status = entity.StatusInProgress
	found.CurrentPhase = entity.PhaseDoing
	originalVersion := found.Version
	if err := repo.Update(ctx, found); err != nil {
		t.Fatalf("update: %v", err)
	}
	if found.Version != originalVersion+1 {
		t.Errorf("version = %d, want %d", found.Version, originalVersion+1)
	}

	// Concurrent modification should fail
	stale := &entity.Task{
		ID:           "task-1",
		Title:        "Stale",
		CurrentPhase: entity.PhasePlanning,
		Status:       entity.StatusPending,
		Version:      originalVersion, // stale version
		UpdatedAt:    time.Now(),
	}
	err = repo.Update(ctx, stale)
	if err == nil {
		t.Error("expected concurrent modification error")
	}

	// FindByFilters
	tasks, err := repo.FindByFilters(ctx, repository.Criteria{})
	if err != nil {
		t.Fatalf("find by filters: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	// Delete
	if err := repo.Delete(ctx, "task-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = repo.Find(ctx, "task-1")
	if err == nil {
		t.Error("expected not found after delete")
	}
}

func TestTaskRepositoryFindByFiltersWithCriteria(t *testing.T) {
	db := setupRepoTestDB(t)
	defer db.Close()

	repo := NewTaskRepositorySQLite(db)
	ctx := context.Background()

	now := time.Now()
	tasks := []*entity.Task{
		{ID: "t1", Title: "A", CurrentPhase: entity.PhasePlanning, Status: entity.StatusPending, Version: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "t2", Title: "B", CurrentPhase: entity.PhaseDoing, Status: entity.StatusInProgress, Version: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "t3", Title: "C", CurrentPhase: entity.PhasePlanning, Status: entity.StatusCompleted, Version: 1, CreatedAt: now, UpdatedAt: now},
	}
	for _, task := range tasks {
		if err := repo.Create(ctx, task); err != nil {
			t.Fatalf("create %s: %v", task.ID, err)
		}
	}

	// Filter by phase = planning
	result, err := repo.FindByFilters(ctx, repository.Criteria{
		{Key: "current_phase", Value: "planning", Operator: repository.OpEquals},
	})
	if err != nil {
		t.Fatalf("find by filters: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 planning tasks, got %d", len(result))
	}
}