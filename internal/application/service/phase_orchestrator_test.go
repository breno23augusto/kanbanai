package service

import (
	"context"
	"sync"
	"testing"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/repository"
	"time"
)

// reuse fakes from usecase_test.go via a local copy to avoid import cycle.
// We define minimal fakes here since we're in the service package.

type fakeTaskRepoSvc struct {
	mu    sync.Mutex
	tasks map[string]*entity.Task
}

func newFakeTaskRepoSvc() *fakeTaskRepoSvc {
	return &fakeTaskRepoSvc{tasks: make(map[string]*entity.Task)}
}

func (r *fakeTaskRepoSvc) Create(ctx context.Context, task *entity.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[task.ID] = task
	return nil
}

func (r *fakeTaskRepoSvc) Update(ctx context.Context, task *entity.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.tasks[task.ID]
	if !ok {
		return nil
	}
	if existing.Version != task.Version {
		return nil
	}
	task.Version++
	r.tasks[task.ID] = task
	return nil
}

func (r *fakeTaskRepoSvc) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tasks, id)
	return nil
}

func (r *fakeTaskRepoSvc) Find(ctx context.Context, id string) (*entity.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return nil, nil
	}
	copied := *t
	return &copied, nil
}

func (r *fakeTaskRepoSvc) FindByFilters(ctx context.Context, criteria repository.Criteria) ([]*entity.Task, error) {
	return nil, nil
}

type fakePhaseOutputRepoSvc struct{}

func (r *fakePhaseOutputRepoSvc) Create(ctx context.Context, output *entity.PhaseOutput) error { return nil }
func (r *fakePhaseOutputRepoSvc) Update(ctx context.Context, output *entity.PhaseOutput) error { return nil }
func (r *fakePhaseOutputRepoSvc) Delete(ctx context.Context, id string) error                  { return nil }
func (r *fakePhaseOutputRepoSvc) Find(ctx context.Context, id string) (*entity.PhaseOutput, error) {
	return nil, nil
}
func (r *fakePhaseOutputRepoSvc) FindByFilters(ctx context.Context, criteria repository.Criteria) ([]*entity.PhaseOutput, error) {
	return nil, nil
}

type fakeHarness struct {
	dispatchedPhases []entity.Phase
	dispatchedTasks  []string
	killedTasks      []string
	mu               sync.Mutex
}

func (h *fakeHarness) Dispatch(ctx context.Context, task *entity.Task, phase entity.Phase, prompt string) error {
	h.mu.Lock()
	h.dispatchedPhases = append(h.dispatchedPhases, phase)
	h.dispatchedTasks = append(h.dispatchedTasks, task.ID)
	h.mu.Unlock()
	return nil
}

func (h *fakeHarness) KillProcess(taskID string) {
	h.mu.Lock()
	h.killedTasks = append(h.killedTasks, taskID)
	h.mu.Unlock()
}

type recordingDisp struct {
	events []event.Event
	mu     sync.Mutex
}

func (d *recordingDisp) Subscribe(event.EventType, event.Handler) {}
func (d *recordingDisp) SubscribeAll(event.Handler)                 {}
func (d *recordingDisp) Publish(evt event.Event) {
	d.mu.Lock()
	d.events = append(d.events, evt)
	d.mu.Unlock()
}

func (d *recordingDisp) getEvents() []event.Event {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.events
}

func TestOrchestratorStartFlow(t *testing.T) {
	repo := newFakeTaskRepoSvc()
	harness := &fakeHarness{}
	disp := &recordingDisp{}
	pb := NewPromptBuilder()

	orch := NewPhaseOrchestrator(repo, &fakePhaseOutputRepoSvc{}, harness, pb, disp)

	task := &entity.Task{
		ID:           "t1",
		Title:        "Test",
		CurrentPhase: entity.PhasePlanning,
		Status:       entity.StatusPending,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.tasks["t1"] = task

	err := orch.StartFlow(context.Background(), task)
	if err != nil {
		t.Fatalf("StartFlow error: %v", err)
	}

	harness.mu.Lock()
	if len(harness.dispatchedPhases) != 1 || harness.dispatchedPhases[0] != entity.PhasePlanning {
		t.Errorf("expected planning dispatch, got %v", harness.dispatchedPhases)
	}
	harness.mu.Unlock()

	events := disp.getEvents()
	foundPlanningStarted := false
	for _, e := range events {
		if e.Type == event.PhasePlanningStarted {
			foundPlanningStarted = true
		}
	}
	if !foundPlanningStarted {
		t.Error("expected PhasePlanningStarted event")
	}
}

func TestOrchestratorAdvancePhaseTransitionsLane(t *testing.T) {
	repo := newFakeTaskRepoSvc()
	harness := &fakeHarness{}
	disp := &recordingDisp{}
	pb := NewPromptBuilder()

	orch := NewPhaseOrchestrator(repo, &fakePhaseOutputRepoSvc{}, harness, pb, disp)

	task := &entity.Task{
		ID:           "t1",
		Title:        "Test",
		CurrentPhase: entity.PhasePlanning,
		Status:       entity.StatusInProgress,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.tasks["t1"] = task

	err := orch.AdvancePhase(context.Background(), "t1")
	if err != nil {
		t.Fatalf("AdvancePhase error: %v", err)
	}

	updated := repo.tasks["t1"]
	if updated.CurrentPhase != entity.PhaseTodo {
		t.Errorf("phase = %s, want todo", updated.CurrentPhase)
	}

	// Should have dispatched the next phase (todo)
	harness.mu.Lock()
	if len(harness.dispatchedPhases) != 1 || harness.dispatchedPhases[0] != entity.PhaseTodo {
		t.Errorf("expected todo dispatch, got %v", harness.dispatchedPhases)
	}
	harness.mu.Unlock()

	// Should have emitted LaneTransitionCompleted with correct from/to
	events := disp.getEvents()
	foundTransition := false
	for _, e := range events {
		if e.Type == event.LaneTransitionCompleted {
			from, _ := e.Payload.(map[string]any)["from"]
			to, _ := e.Payload.(map[string]any)["to"]
			if from == entity.PhasePlanning && to == entity.PhaseTodo {
				foundTransition = true
			}
		}
	}
	if !foundTransition {
		t.Error("expected LaneTransitionCompleted event from planning to todo")
	}
}

func TestOrchestratorAdvancePhaseReachesDone(t *testing.T) {
	repo := newFakeTaskRepoSvc()
	harness := &fakeHarness{}
	disp := &recordingDisp{}
	pb := NewPromptBuilder()

	orch := NewPhaseOrchestrator(repo, &fakePhaseOutputRepoSvc{}, harness, pb, disp)

	task := &entity.Task{
		ID:           "t1",
		Title:        "Test",
		CurrentPhase: entity.PhaseTesting,
		Status:       entity.StatusInProgress,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.tasks["t1"] = task

	err := orch.AdvancePhase(context.Background(), "t1")
	if err != nil {
		t.Fatalf("AdvancePhase error: %v", err)
	}

	updated := repo.tasks["t1"]
	if updated.CurrentPhase != entity.PhaseDone {
		t.Errorf("phase = %s, want done", updated.CurrentPhase)
	}
	if updated.Status != entity.StatusCompleted {
		t.Errorf("status = %s, want completed", updated.Status)
	}

	events := disp.getEvents()
	foundDone := false
	for _, e := range events {
		if e.Type == event.PhaseDoneReached {
			foundDone = true
		}
	}
	if !foundDone {
		t.Error("expected PhaseDoneReached event")
	}

	// Should NOT dispatch another phase since done is terminal
	harness.mu.Lock()
	if len(harness.dispatchedPhases) != 0 {
		t.Errorf("expected no dispatch at done, got %v", harness.dispatchedPhases)
	}
	harness.mu.Unlock()
}

func TestOrchestratorKillProcess(t *testing.T) {
	harness := &fakeHarness{}
	orch := NewPhaseOrchestrator(newFakeTaskRepoSvc(), &fakePhaseOutputRepoSvc{}, harness, NewPromptBuilder(), &recordingDisp{})

	orch.KillProcess("t1")

	harness.mu.Lock()
	if len(harness.killedTasks) != 1 || harness.killedTasks[0] != "t1" {
		t.Errorf("expected kill for t1, got %v", harness.killedTasks)
	}
	harness.mu.Unlock()
}