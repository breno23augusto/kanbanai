package harness

import (
	"context"
	"fmt"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
	"sync"
	"os/exec"
)

type Adapter struct {
	configs         map[entity.Phase]entity.PhaseConfig
	builder         *CommandBuilder
	dispatcher      event.Dispatcher
	processRegistry map[string]*exec.Cmd
	mu              sync.RWMutex
}

func NewAdapter(
	configs map[entity.Phase]entity.PhaseConfig,
	mcpPort string,
	dispatcher event.Dispatcher,
) *Adapter {
	return &Adapter{
		configs:         configs,
		builder:         NewCommandBuilder(mcpPort),
		dispatcher:      dispatcher,
		processRegistry: make(map[string]*exec.Cmd),
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

func (a *Adapter) KillProcess(taskID string) {
	a.mu.Lock()
	cmd, exists := a.processRegistry[taskID]
	if exists {
		delete(a.processRegistry, taskID)
	}
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
		Type:    event.HarnessCommandDispatched,
		TaskID:  task.ID,
		Payload: map[string]any{"phase": phase, "model": config.ModelName},
	})

	cmd, err := a.builder.Build(ctx, config.HarnessCmd, config.ModelName, task.ID, prompt)
	if err != nil {
		return fmt.Errorf("failed to build command: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start harness: %w", err)
	}

	a.RegisterProcess(task.ID, cmd)
	go a.monitorProcess(cmd, task.ID, phase, config)
	return nil
}

func (a *Adapter) monitorProcess(cmd *exec.Cmd, taskID string, phase entity.Phase, config entity.PhaseConfig) {
	defer a.UnregisterProcess(taskID)

	err := cmd.Wait()
	if err != nil {
		a.dispatcher.Publish(event.Event{
			Type:    event.HarnessErrorOccurred,
			TaskID:  taskID,
			Payload: map[string]any{"phase": phase, "error": err.Error()},
		})
	}
}
