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
	pb := NewPromptBuilder("http://localhost:8080/api/v1")

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
	pb := NewPromptBuilder("http://localhost:8080/api/v1")

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
	pb := NewPromptBuilder("http://localhost:8080/api/v1")

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
	orch := NewPhaseOrchestrator(newFakeTaskRepoSvc(), &fakePhaseOutputRepoSvc{}, harness, NewPromptBuilder("http://localhost:8080/api/v1"), &recordingDisp{})

	orch.KillProcess("t1")

	harness.mu.Lock()
	if len(harness.killedTasks) != 1 || harness.killedTasks[0] != "t1" {
		t.Errorf("expected kill for t1, got %v", harness.killedTasks)
	}
	harness.mu.Unlock()
}
func TestOrchestratorRestartPhaseRedpatchesAndResetsAttempts(t *testing.T) {
	repo := newFakeTaskRepoSvc()
	harness := &fakeHarness{}
	disp := &recordingDisp{}
	pb := NewPromptBuilder("http://localhost:8080/api/v1")

	orch := NewPhaseOrchestrator(repo, &fakePhaseOutputRepoSvc{}, harness, pb, disp)

	task := &entity.Task{
		ID:           "t1",
		Title:        "Test",
		CurrentPhase: entity.PhaseDoing,
		Status:       entity.StatusFailed,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.tasks["t1"] = task

	if err := orch.RestartPhase(context.Background(), "t1"); err != nil {
		t.Fatalf("RestartPhase error: %v", err)
	}

	// Status should move back to in_progress and the current phase re-dispatched.
	updated := repo.tasks["t1"]
	if updated.Status != entity.StatusInProgress {
		t.Errorf("status = %s, want in_progress", updated.Status)
	}

	harness.mu.Lock()
	if len(harness.dispatchedPhases) != 1 || harness.dispatchedPhases[0] != entity.PhaseDoing {
		t.Errorf("expected doing re-dispatch, got %v", harness.dispatchedPhases)
	}
	harness.mu.Unlock()

	// Retry counter should be reset to 0 (a subsequent failure starts at attempt 1).
	orch.HandleRetry(context.Background(), "t1", entity.PhaseDoing, 0, "")
	orch.mu.Lock()
	if orch.retryAttempts["t1"] != 1 {
		t.Errorf("retryAttempts = %d, want 1 after reset + one failure", orch.retryAttempts["t1"])
	}
	orch.mu.Unlock()
}

func TestOrchestratorPauseTaskKillsProcessAndMarksPaused(t *testing.T) {
	repo := newFakeTaskRepoSvc()
	harness := &fakeHarness{}
	disp := &recordingDisp{}
	orch := NewPhaseOrchestrator(repo, &fakePhaseOutputRepoSvc{}, harness, NewPromptBuilder("http://localhost:8080/api/v1"), disp)

	task := &entity.Task{
		ID:           "t1",
		Title:        "Test",
		CurrentPhase: entity.PhaseDoing,
		Status:       entity.StatusInProgress,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.tasks["t1"] = task

	if err := orch.PauseTask(context.Background(), "t1"); err != nil {
		t.Fatalf("PauseTask error: %v", err)
	}

	// Harness process must be killed.
	harness.mu.Lock()
	if len(harness.killedTasks) != 1 || harness.killedTasks[0] != "t1" {
		t.Errorf("expected kill for t1, got %v", harness.killedTasks)
	}
	harness.mu.Unlock()

	// Status must move to paused; phase is preserved.
	updated := repo.tasks["t1"]
	if updated.Status != entity.StatusPaused {
		t.Errorf("status = %s, want paused", updated.Status)
	}
	if updated.CurrentPhase != entity.PhaseDoing {
		t.Errorf("phase = %s, want doing (unchanged)", updated.CurrentPhase)
	}

	// A TaskPaused event must be emitted.
	found := false
	for _, e := range disp.getEvents() {
		if e.Type == event.TaskPaused {
			found = true
		}
	}
	if !found {
		t.Error("expected TaskPaused event")
	}
}

func TestOrchestratorPauseTaskRejectsNonRunningTask(t *testing.T) {
	repo := newFakeTaskRepoSvc()
	harness := &fakeHarness{}
	orch := NewPhaseOrchestrator(repo, &fakePhaseOutputRepoSvc{}, harness, NewPromptBuilder("http://localhost:8080/api/v1"), &recordingDisp{})

	repo.tasks["t1"] = &entity.Task{
		ID:           "t1",
		CurrentPhase: entity.PhasePlanning,
		Status:       entity.StatusPending,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := orch.PauseTask(context.Background(), "t1"); err == nil {
		t.Fatal("expected error pausing a non-running task, got nil")
	}

	harness.mu.Lock()
	if len(harness.killedTasks) != 0 {
		t.Errorf("expected no kill, got %v", harness.killedTasks)
	}
	harness.mu.Unlock()
}

func TestOrchestratorResumeTaskRedispatchesAndEmitsEvent(t *testing.T) {
	repo := newFakeTaskRepoSvc()
	harness := &fakeHarness{}
	disp := &recordingDisp{}
	orch := NewPhaseOrchestrator(repo, &fakePhaseOutputRepoSvc{}, harness, NewPromptBuilder("http://localhost:8080/api/v1"), disp)

	task := &entity.Task{
		ID:           "t1",
		Title:        "Test",
		CurrentPhase: entity.PhaseDoing,
		Status:       entity.StatusPaused,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.tasks["t1"] = task

	if err := orch.ResumeTask(context.Background(), "t1"); err != nil {
		t.Fatalf("ResumeTask error: %v", err)
	}

	// dispatchPhase sets status to in_progress and re-dispatches the phase.
	updated := repo.tasks["t1"]
	if updated.Status != entity.StatusInProgress {
		t.Errorf("status = %s, want in_progress", updated.Status)
	}

	harness.mu.Lock()
	if len(harness.dispatchedPhases) != 1 || harness.dispatchedPhases[0] != entity.PhaseDoing {
		t.Errorf("expected doing re-dispatch, got %v", harness.dispatchedPhases)
	}
	harness.mu.Unlock()

	// A TaskResumed event must be emitted.
	found := false
	for _, e := range disp.getEvents() {
		if e.Type == event.TaskResumed {
			found = true
		}
	}
	if !found {
		t.Error("expected TaskResumed event")
	}
}

func TestOrchestratorResumeTaskRejectsNonPausedTask(t *testing.T) {
	repo := newFakeTaskRepoSvc()
	harness := &fakeHarness{}
	orch := NewPhaseOrchestrator(repo, &fakePhaseOutputRepoSvc{}, harness, NewPromptBuilder("http://localhost:8080/api/v1"), &recordingDisp{})

	repo.tasks["t1"] = &entity.Task{
		ID:           "t1",
		CurrentPhase: entity.PhaseDoing,
		Status:       entity.StatusInProgress,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := orch.ResumeTask(context.Background(), "t1"); err == nil {
		t.Fatal("expected error resuming a non-paused task, got nil")
	}

	harness.mu.Lock()
	if len(harness.dispatchedPhases) != 0 {
		t.Errorf("expected no dispatch, got %v", harness.dispatchedPhases)
	}
	harness.mu.Unlock()
}

func TestOrchestratorReopenPhaseMovesBackAndRedispatches(t *testing.T) {
	repo := newFakeTaskRepoSvc()
	harness := &fakeHarness{}
	disp := &recordingDisp{}
	orch := NewPhaseOrchestrator(repo, &fakePhaseOutputRepoSvc{}, harness, NewPromptBuilder("http://localhost:8080/api/v1"), disp)

	repo.tasks["t1"] = &entity.Task{
		ID:           "t1",
		Title:        "tic tac toe",
		CurrentPhase: entity.PhaseValidating,
		Status:       entity.StatusInProgress,
		ErrorMessage: "stale",
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := orch.ReopenPhase(context.Background(), "t1", entity.PhaseDoing, "criteria X not met"); err != nil {
		t.Fatalf("ReopenPhase error: %v", err)
	}

	updated := repo.tasks["t1"]
	if updated.CurrentPhase != entity.PhaseDoing {
		t.Errorf("phase = %s, want doing", updated.CurrentPhase)
	}
	if updated.ErrorMessage != "" {
		t.Errorf("error message should be cleared on reopen, got %q", updated.ErrorMessage)
	}

	harness.mu.Lock()
	if len(harness.dispatchedPhases) != 1 || harness.dispatchedPhases[0] != entity.PhaseDoing {
		t.Errorf("expected doing re-dispatch, got %v", harness.dispatchedPhases)
	}
	harness.mu.Unlock()

	found := false
	for _, e := range disp.getEvents() {
		if e.Type == event.LaneReopened {
			from, _ := e.Payload.(map[string]any)["from"]
			to, _ := e.Payload.(map[string]any)["to"]
			if from == entity.PhaseValidating && to == entity.PhaseDoing {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected LaneReopened event from validating to doing")
	}
}

func TestOrchestratorReopenPhaseRejectsNonPrecedingTarget(t *testing.T) {
	repo := newFakeTaskRepoSvc()
	harness := &fakeHarness{}
	orch := NewPhaseOrchestrator(repo, &fakePhaseOutputRepoSvc{}, harness, NewPromptBuilder("http://localhost:8080/api/v1"), &recordingDisp{})

	repo.tasks["t1"] = &entity.Task{
		ID:           "t1",
		CurrentPhase: entity.PhaseValidating,
		Status:       entity.StatusInProgress,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// target == current: not allowed (use RestartPhase for same-phase reruns).
	if err := orch.ReopenPhase(context.Background(), "t1", entity.PhaseValidating, ""); err == nil {
		t.Fatal("expected error reopening to the current phase, got nil")
	}
	// target later than current: not allowed.
	if err := orch.ReopenPhase(context.Background(), "t1", entity.PhaseTesting, ""); err == nil {
		t.Fatal("expected error reopening to a later phase, got nil")
	}
	// terminal target: not allowed.
	if err := orch.ReopenPhase(context.Background(), "t1", entity.PhaseDone, ""); err == nil {
		t.Fatal("expected error reopening to terminal phase, got nil")
	}

	harness.mu.Lock()
	if len(harness.dispatchedPhases) != 0 {
		t.Errorf("expected no dispatch on rejected reopen, got %v", harness.dispatchedPhases)
	}
	harness.mu.Unlock()
}

func TestOrchestratorReopenPhaseRejectsInactiveTask(t *testing.T) {
	repo := newFakeTaskRepoSvc()
	harness := &fakeHarness{}
	orch := NewPhaseOrchestrator(repo, &fakePhaseOutputRepoSvc{}, harness, NewPromptBuilder("http://localhost:8080/api/v1"), &recordingDisp{})

	repo.tasks["t1"] = &entity.Task{
		ID:           "t1",
		CurrentPhase: entity.PhaseValidating,
		Status:       entity.StatusFailed,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := orch.ReopenPhase(context.Background(), "t1", entity.PhaseDoing, ""); err == nil {
		t.Fatal("expected error reopening a failed task, got nil")
	}
}
