package harness

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"kanbanai/internal/adapter/out/livetail"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/port"
)

type Adapter struct {
	provider        port.PhaseConfigProvider
	builder         *CommandBuilder
	dispatcher      event.Dispatcher
	liveStore       *livetail.Store
	processRegistry map[string]*exec.Cmd
	killed          map[string]struct{}
	mu              sync.RWMutex
}

func NewAdapter(
	provider port.PhaseConfigProvider,
	mcpPort string,
	apiBaseURL string,
	dispatcher event.Dispatcher,
	liveStore *livetail.Store,
) *Adapter {
	return &Adapter{
		provider:        provider,
		builder:         NewCommandBuilder(mcpPort, apiBaseURL),
		dispatcher:      dispatcher,
		liveStore:       liveStore,
		processRegistry: make(map[string]*exec.Cmd),
		killed:          make(map[string]struct{}),
	}
}

func (a *Adapter) RegisterProcess(taskID string, cmd *exec.Cmd) {
	a.mu.Lock()
	a.processRegistry[taskID] = cmd
	a.mu.Unlock()
}

func (a *Adapter) UnregisterProcess(taskID string) {
	a.mu.Lock()
	delete(a.processRegistry, taskID)
	a.mu.Unlock()
}

func (a *Adapter) GetProcess(taskID string) *exec.Cmd {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.processRegistry[taskID]
}

// KillProcess terminates a running harness process for the given task. The
// termination is marked so monitorProcess does not treat it as a harness
// failure and trigger an automatic retry (SPEC §32.3).
func (a *Adapter) KillProcess(taskID string) {
	a.mu.Lock()
	cmd, exists := a.processRegistry[taskID]
	if exists {
		delete(a.processRegistry, taskID)
	}
	a.killed[taskID] = struct{}{}
	a.mu.Unlock()

	if exists && cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func (a *Adapter) Dispatch(ctx context.Context, task *entity.Task, phase entity.Phase, prompt string) error {
	config := a.provider.Get(phase)
	a.dispatcher.Publish(event.Event{
		Type:      event.HarnessCommandDispatched,
		TaskID:    task.ID,
		Payload:   map[string]any{"phase": phase, "model": config.ModelName},
		Timestamp: time.Now(),
	})

	// Enforce the per-phase timeout from configuration. The harness process
	// outlives the request/subscriber that triggered the dispatch, so the
	// deadline is derived from the background context — only TimeoutSec and an
	// explicit KillProcess terminate it (SPEC §13.2 / §32.3).
	timeout := time.Duration(config.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(context.Background(), timeout)

	cmd, err := a.builder.Build(runCtx, config.HarnessCmd, config.ModelName, task.ID, string(phase), task.Workspace, prompt)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to build command: %w", err)
	}

	// Open a fresh live stream for this run so the operator can watch the
	// agent work in real time (each Dispatch resets the tail). The bytes.Buffer
	// is kept for the failure-reason tail; livetail mirrors the same bytes live.
	liveStream := a.liveStore.Open(task.ID)
	output := &bytes.Buffer{}
	cmd.Stdout = io.MultiWriter(output, liveStream)
	cmd.Stderr = io.MultiWriter(output, liveStream)

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start harness: %w", err)
	}

	a.RegisterProcess(task.ID, cmd)
	go a.monitorProcess(cmd, cancel, task.ID, phase, output, liveStream)
	return nil
}

func (a *Adapter) monitorProcess(cmd *exec.Cmd, cancel context.CancelFunc, taskID string, phase entity.Phase, output *bytes.Buffer, liveStream *livetail.Stream) {
	defer a.UnregisterProcess(taskID)
	defer cancel()
	defer liveStream.End()

	err := cmd.Wait()

	a.mu.Lock()
	_, killed := a.killed[taskID]
	if killed {
		delete(a.killed, taskID)
	}
	a.mu.Unlock()

	// Process was intentionally killed (task deletion) — do not retry.
	if killed {
		return
	}

	// Non-zero exit or timeout: publish a harness error so the orchestrator
	// can apply the retry policy (SPEC §13.2 / §32.3). A clean exit (err == nil)
	// without complete_phase being called leaves the task in progress; that is
	// a harness misbehavior and is not retried automatically.
	if err != nil {
		// The captured output is what makes a failure debuggable: it contains
		// the harness wrapper's own diagnostics (e.g. "agent prompt failed: …",
		// "complete failed: 404 …"). Trim to the tail to bound storage.
		tail := output.String()
		if len(tail) > 4000 {
			tail = tail[len(tail)-4000:]
		}
		a.dispatcher.Publish(event.Event{
			Type:      event.HarnessErrorOccurred,
			TaskID:    taskID,
			Payload:   map[string]any{"phase": phase, "error": err.Error(), "output": tail},
			Timestamp: time.Now(),
		})
	}
}
