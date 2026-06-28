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

// statefulPhaseOutputRepo is a fake that actually stores phase outputs so the
// orchestrator can exercise populatePriorContext end-to-end.
type statefulPhaseOutputRepo struct {
	mu      sync.Mutex
	outputs map[string]*entity.PhaseOutput // keyed by id
}

func newStatefulPhaseOutputRepo() *statefulPhaseOutputRepo {
	return &statefulPhaseOutputRepo{outputs: make(map[string]*entity.PhaseOutput)}
}

func (r *statefulPhaseOutputRepo) Create(ctx context.Context, output *entity.PhaseOutput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.outputs[output.ID] = output
	return nil
}
func (r *statefulPhaseOutputRepo) Update(ctx context.Context, output *entity.PhaseOutput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.outputs[output.ID] = output
	return nil
}
func (r *statefulPhaseOutputRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.outputs, id)
	return nil
}
func (r *statefulPhaseOutputRepo) Find(ctx context.Context, id string) (*entity.PhaseOutput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	po, ok := r.outputs[id]
	if !ok {
		return nil, nil
	}
	cp := *po
	return &cp, nil
}
func (r *statefulPhaseOutputRepo) FindByFilters(ctx context.Context, criteria repository.Criteria) ([]*entity.PhaseOutput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var taskID string
	for _, c := range criteria {
		if c.Key == "task_id" {
			if s, ok := c.Value.(string); ok {
				taskID = s
			}
		}
	}
	var result []*entity.PhaseOutput
	for _, po := range r.outputs {
		if taskID == "" || po.TaskID == taskID {
			cp := *po
			result = append(result, &cp)
		}
	}
	return result, nil
}

// capturingHarness records the prompt sent for each dispatch so tests can
// assert that prior-phase context was injected.
type capturingHarness struct {
	fakeHarness
	prompts   map[entity.Phase]string
	promptMu  sync.Mutex
}

func newCapturingHarness() *capturingHarness {
	return &capturingHarness{prompts: make(map[entity.Phase]string)}
}

func (h *capturingHarness) Dispatch(ctx context.Context, task *entity.Task, phase entity.Phase, prompt string) error {
	h.promptMu.Lock()
	h.prompts[phase] = prompt
	h.promptMu.Unlock()
	return h.fakeHarness.Dispatch(ctx, task, phase, prompt)
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

func TestOrchestratorValidatingPromptCarriesPriorContext(t *testing.T) {
	repo := newFakeTaskRepoSvc()
	outRepo := newStatefulPhaseOutputRepo()
	harness := newCapturingHarness()
	disp := &recordingDisp{}
	pb := NewPromptBuilder("http://localhost:8080/api/v1")

	orch := NewPhaseOrchestrator(repo, outRepo, harness, pb, disp)

	now := time.Now()
	task := &entity.Task{
		ID:           "t1",
		Title:        "tic tac toe",
		Description:  "simple js tic tac toe game",
		CurrentPhase: entity.PhaseDoing,
		Status:       entity.StatusInProgress,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	repo.tasks["t1"] = task

	// Prior-phase artifacts the validation phase must evaluate.
	_ = outRepo.Create(context.Background(), &entity.PhaseOutput{
		ID: "po-plan", TaskID: "t1", Phase: entity.PhasePlanning,
		Output: "AC1: 3x3 grid board", CreatedAt: now, UpdatedAt: now,
	})
	_ = outRepo.Create(context.Background(), &entity.PhaseOutput{
		ID: "po-todo", TaskID: "t1", Phase: entity.PhaseTodo,
		Output: "AC2: two players alternate turns", CreatedAt: now, UpdatedAt: now,
	})
	_ = outRepo.Create(context.Background(), &entity.PhaseOutput{
		ID: "po-doing", TaskID: "t1", Phase: entity.PhaseDoing,
		Output: "Implemented index.html with board + click handlers", CreatedAt: now, UpdatedAt: now,
	})

	if err := orch.AdvancePhase(context.Background(), "t1"); err != nil {
		t.Fatalf("AdvancePhase error: %v", err)
	}

	if updated := repo.tasks["t1"]; updated.CurrentPhase != entity.PhaseValidating {
		t.Fatalf("phase = %s, want validating", updated.CurrentPhase)
	}

	harness.promptMu.Lock()
	prompt := harness.prompts[entity.PhaseValidating]
	harness.promptMu.Unlock()

	if prompt == "" {
		t.Fatalf("no prompt captured for validating phase")
	}

	mustContain := []string{
		"ORIGINAL PROMPT",
		"simple js tic tac toe game",
		"ACCEPTANCE CRITERIA",
		"AC1: 3x3 grid board",
		"AC2: two players alternate turns",
		"IMPLEMENTATION REPORT",
		"Implemented index.html with board + click handlers",
		"STEP 1",
		"STEP 2",
		"STEP 3",
		"VERDICT",
	}
	for _, sub := range mustContain {
		if !contains(prompt, sub) {
			t.Errorf("validating prompt missing %q", sub)
		}
	}
}

func TestOrchestratorValidatingPromptHandlesMissingOutputs(t *testing.T) {
	repo := newFakeTaskRepoSvc()
	outRepo := newStatefulPhaseOutputRepo()
	harness := newCapturingHarness()
	disp := &recordingDisp{}
	pb := NewPromptBuilder("http://localhost:8080/api/v1")

	orch := NewPhaseOrchestrator(repo, outRepo, harness, pb, disp)

	now := time.Now()
	task := &entity.Task{
		ID:           "t1",
		Title:        "tic tac toe",
		Description:  "simple js tic tac toe game",
		CurrentPhase: entity.PhaseDoing,
		Status:       entity.StatusInProgress,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	repo.tasks["t1"] = task

	// No prior outputs saved at all — the prompt must still render and instruct
	// the reviewer to fetch them via get_task.
	if err := orch.AdvancePhase(context.Background(), "t1"); err != nil {
		t.Fatalf("AdvancePhase error: %v", err)
	}

	harness.promptMu.Lock()
	prompt := harness.prompts[entity.PhaseValidating]
	harness.promptMu.Unlock()

	if prompt == "" {
		t.Fatalf("no prompt captured for validating phase")
	}
	if !contains(prompt, "(no output saved)") {
		t.Errorf("expected placeholder for missing outputs, got:\n%s", prompt)
	}
	if !contains(prompt, "get_task") {
		t.Errorf("expected get_task fallback instruction when outputs missing")
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
