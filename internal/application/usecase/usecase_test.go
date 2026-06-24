package usecase

import (
	"context"
	"fmt"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/repository"
	"kanbanai/internal/application/dto"
	"sync"
	"testing"
	"time"
)

// --- Fakes ---

type fakeTaskRepo struct {
	mu    sync.Mutex
	tasks map[string]*entity.Task
}

func newFakeTaskRepo() *fakeTaskRepo {
	return &fakeTaskRepo{tasks: make(map[string]*entity.Task)}
}

func (r *fakeTaskRepo) Create(ctx context.Context, task *entity.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[task.ID] = task
	return nil
}

func (r *fakeTaskRepo) Update(ctx context.Context, task *entity.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.tasks[task.ID]
	if !ok || existing.Version != task.Version {
		return fmt.Errorf("concurrent modification: version mismatch")
	}
	task.Version++
	r.tasks[task.ID] = task
	return nil
}

func (r *fakeTaskRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tasks, id)
	return nil
}

func (r *fakeTaskRepo) Find(ctx context.Context, id string) (*entity.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	// return a copy
	copied := *t
	return &copied, nil
}

func (r *fakeTaskRepo) FindByFilters(ctx context.Context, criteria repository.Criteria) ([]*entity.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*entity.Task
	for _, t := range r.tasks {
		result = append(result, t)
	}
	return result, nil
}

type fakePhaseOutputRepo struct {
	outputs map[string]*entity.PhaseOutput
	mu      sync.Mutex
}

func newFakePhaseOutputRepo() *fakePhaseOutputRepo {
	return &fakePhaseOutputRepo{outputs: make(map[string]*entity.PhaseOutput)}
}

func (r *fakePhaseOutputRepo) Create(ctx context.Context, output *entity.PhaseOutput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.outputs[output.ID] = output
	return nil
}

func (r *fakePhaseOutputRepo) Update(ctx context.Context, output *entity.PhaseOutput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.outputs[output.ID] = output
	return nil
}

func (r *fakePhaseOutputRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.outputs, id)
	return nil
}

func (r *fakePhaseOutputRepo) Find(ctx context.Context, id string) (*entity.PhaseOutput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, ok := r.outputs[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return o, nil
}

func (r *fakePhaseOutputRepo) FindByFilters(ctx context.Context, criteria repository.Criteria) ([]*entity.PhaseOutput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*entity.PhaseOutput
	for _, o := range r.outputs {
		result = append(result, o)
	}
	return result, nil
}

type recordingDispatcher struct {
	events []event.Event
	mu     sync.Mutex
}

func (d *recordingDispatcher) Subscribe(event.EventType, event.Handler) {}
func (d *recordingDispatcher) SubscribeAll(event.Handler)                 {}
func (d *recordingDispatcher) Publish(evt event.Event) {
	d.mu.Lock()
	d.events = append(d.events, evt)
	d.mu.Unlock()
}

func (d *recordingDispatcher) getEvents() []event.Event {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.events
}

// --- Tests ---

func TestCreateTask(t *testing.T) {
	repo := newFakeTaskRepo()
	disp := &recordingDispatcher{}
	uc := NewCreateTask(repo, disp)

	result, err := uc.Execute(context.Background(), dto.CreateTaskInput{
		Title:       "My Task",
		Description: "A description",
		Priority:    3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Title != "My Task" {
		t.Errorf("title = %s, want My Task", result.Title)
	}
	if result.CurrentPhase != entity.PhasePlanning {
		t.Errorf("phase = %s, want planning", result.CurrentPhase)
	}
	if result.Status != entity.StatusPending {
		t.Errorf("status = %s, want pending", result.Status)
	}
	if result.Version != 1 {
		t.Errorf("version = %d, want 1", result.Version)
	}

	events := disp.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != event.TaskCreated {
		t.Errorf("event type = %s, want task.created", events[0].Type)
	}
}

func TestCreateTaskEmptyTitle(t *testing.T) {
	repo := newFakeTaskRepo()
	disp := &recordingDispatcher{}
	uc := NewCreateTask(repo, disp)

	_, err := uc.Execute(context.Background(), dto.CreateTaskInput{Title: ""})
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestAdvancePhaseMarksPhaseCompleted(t *testing.T) {
	repo := newFakeTaskRepo()
	disp := &recordingDispatcher{}

	task := &entity.Task{
		ID:           "t1",
		Title:        "Task",
		CurrentPhase: entity.PhasePlanning,
		Status:       entity.StatusInProgress,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.tasks["t1"] = task

	phaseOutputRepo := newFakePhaseOutputRepo()
	uc := NewAdvancePhase(repo, phaseOutputRepo, disp)

	err := uc.Execute(context.Background(), "t1", entity.PhasePlanning, "planning done")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := disp.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != event.PhasePlanningCompleted {
		t.Errorf("event type = %s, want phase.planning.completed", events[0].Type)
	}

	// Task should NOT be marked as completed (only the phase is done)
	updated := repo.tasks["t1"]
	if updated.Status == entity.StatusCompleted {
		t.Error("task should not be marked completed when a phase completes")
	}
}