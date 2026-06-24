package harness

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
)

type Adapter struct {
	configs         map[entity.Phase]entity.PhaseConfig
	builder         *CommandBuilder
	dispatcher      event.Dispatcher
	processRegistry map[string]*exec.Cmd
	killed          map[string]struct{}
	mu              sync.RWMutex
}

func NewAdapter(
	configs map[entity.Phase]entity.PhaseConfig,
	mcpPort string,
	apiBaseURL string,
	dispatcher event.Dispatcher,
) *Adapter {
	return &Adapter{
		configs:         configs,
		builder:         NewCommandBuilder(mcpPort, apiBaseURL),
		dispatcher:      dispatcher,
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
	config, ok := a.configs[phase]
	if !ok {
		return fmt.Errorf("no config for phase: %s", phase)
	}

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

	cmd, err := a.builder.Build(runCtx, config.HarnessCmd, config.ModelName, task.ID, prompt)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to build command: %w", err)
	}

	// Capture the harness stdout/stderr so a non-zero exit (or timeout) carries
	// a human-readable reason instead of a bare "exit status 1". Without this
	// the frontend has nothing to explain why a phase failed (SPEC §32.3).
	output := &bytes.Buffer{}
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start harness: %w", err)
	}

	a.RegisterProcess(task.ID, cmd)
	go a.monitorProcess(cmd, cancel, task.ID, phase, output)
	return nil
}

func (a *Adapter) monitorProcess(cmd *exec.Cmd, cancel context.CancelFunc, taskID string, phase entity.Phase, output *bytes.Buffer) {
	defer a.UnregisterProcess(taskID)
	defer cancel()

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