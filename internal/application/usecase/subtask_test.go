package usecase

import (
	"context"
	"testing"

	"kanbanai/internal/application/dto"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
)

func TestCreateSubtasksReplacesExisting(t *testing.T) {
	repo := newFakeSubtaskRepo()
	disp := &recordingDispatcher{}
	uc := NewCreateSubtasks(repo, disp)

	// seed a stale subtask that must be cleared on replace
	_ = repo.Create(context.Background(), &entity.Subtask{ID: "stale", TaskID: "t1", Title: "old", Status: entity.SubtaskCompleted})

	result, err := uc.Execute(context.Background(), "t1", []dto.SubtaskInput{
		{Title: "board"}, {Title: "turns"}, {Title: "win check"},
	})
	if err != nil {
		t.Fatalf("CreateSubtasks error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 subtasks, got %d", len(result))
	}
	for i, st := range result {
		if st.Status != entity.SubtaskPending {
			t.Errorf("subtask %d status = %s, want pending", i, st.Status)
		}
		if st.Order != i {
			t.Errorf("subtask %d order = %d, want %d", i, st.Order, i)
		}
	}

	items, _ := repo.FindByTask(context.Background(), "t1")
	if len(items) != 3 {
		t.Errorf("expected 3 persisted subtasks, got %d", len(items))
	}
	if _, err := repo.Find(context.Background(), "stale"); err == nil {
		t.Error("stale subtask should have been deleted on replace")
	}

	// should fire a subtask.created event
	var found bool
	for _, e := range disp.getEvents() {
		if e.Type == event.SubtaskCreated {
			found = true
		}
	}
	if !found {
		t.Error("expected SubtaskCreated event")
	}
}

func TestCreateSubtasksRejectsEmpty(t *testing.T) {
	uc := NewCreateSubtasks(newFakeSubtaskRepo(), &recordingDispatcher{})
	if _, err := uc.Execute(context.Background(), "t1", nil); err == nil {
		t.Error("expected error for empty subtask list")
	}
}

func TestUpdateSubtaskStatus(t *testing.T) {
	repo := newFakeSubtaskRepo()
	disp := &recordingDispatcher{}
	createUC := NewCreateSubtasks(repo, disp)
	created, _ := createUC.Execute(context.Background(), "t1", []dto.SubtaskInput{{Title: "board"}})
	stID := created[0].ID

	updateUC := NewUpdateSubtaskStatus(repo, disp)
	res, err := updateUC.Execute(context.Background(), "t1", stID, entity.SubtaskInProgress)
	if err != nil {
		t.Fatalf("update error: %v", err)
	}
	if res.Status != entity.SubtaskInProgress {
		t.Errorf("status = %s, want in_progress", res.Status)
	}

	res, err = updateUC.Execute(context.Background(), "t1", stID, entity.SubtaskCompleted)
	if err != nil {
		t.Fatalf("update to completed error: %v", err)
	}
	if res.Status != entity.SubtaskCompleted {
		t.Errorf("status = %s, want completed", res.Status)
	}

	// subtask.updated fired for each update
	count := 0
	for _, e := range disp.getEvents() {
		if e.Type == event.SubtaskUpdated {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 SubtaskUpdated events, got %d", count)
	}
}

func TestUpdateSubtaskStatusRejectsWrongTask(t *testing.T) {
	repo := newFakeSubtaskRepo()
	createUC := NewCreateSubtasks(repo, &recordingDispatcher{})
	created, _ := createUC.Execute(context.Background(), "t1", []dto.SubtaskInput{{Title: "x"}})

	updateUC := NewUpdateSubtaskStatus(repo, &recordingDispatcher{})
	if _, err := updateUC.Execute(context.Background(), "other", created[0].ID, entity.SubtaskCompleted); err == nil {
		t.Error("expected error when subtask belongs to a different task")
	}
}

func TestSubtaskSummaryFrom(t *testing.T) {
	items := []entity.Subtask{
		{Status: entity.SubtaskCompleted},
		{Status: entity.SubtaskCompleted},
		{Status: entity.SubtaskInProgress},
		{Status: entity.SubtaskPending},
	}
	s := dto.SubtaskSummaryFrom(items)
	if s.Total != 4 || s.Completed != 2 || s.InProgress != 1 {
		t.Errorf("summary = %+v, want {Total:4, Completed:2, InProgress:1}", s)
	}
}