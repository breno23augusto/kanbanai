package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"kanbanai/config"
	"kanbanai/internal/adapter/in/http/handler"
	eventimpl "kanbanai/internal/adapter/out/event"
	"kanbanai/internal/adapter/out/harness"
	"kanbanai/internal/adapter/out/livetail"
	"kanbanai/internal/adapter/out/persistence/query"
	"kanbanai/internal/adapter/out/persistence/repository"
	"kanbanai/internal/adapter/out/persistence/sqlite"
	"kanbanai/internal/adapter/out/sse"
	"kanbanai/internal/application/service"
	"kanbanai/internal/application/usecase"
	"kanbanai/internal/di"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/event"
)

func Initialize(cfg *config.Config) (*di.Container, error) {
	container := di.NewContainer()

	// 1. Logger
	var logHandler slog.Handler
	if cfg.Log.Level == "debug" {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	logger := slog.New(logHandler)
	slog.SetDefault(logger)
	container.Register("logger", logger)

	// 2. SQLite connection
	db, err := sqlite.NewConnection(cfg.DB.Path)
	if err != nil {
		return nil, err
	}
	container.Register("db", db)

	// 3. Run migrations
	if err := sqlite.RunMigrations(db, cfg.DB.MigrationDir); err != nil {
		return nil, err
	}

	// 4. Repositories and Queries
	taskRepo := repository.NewTaskRepositorySQLite(db)
	eventLogRepo := repository.NewTaskEventLogRepositorySQLite(db)
	phaseOutputRepo := repository.NewPhaseOutputRepositorySQLite(db)
	subtaskRepo := repository.NewSubtaskRepositorySQLite(db)

	container.Register("taskRepo", taskRepo)
	container.Register("eventLogRepo", eventLogRepo)
	container.Register("phaseOutputRepo", phaseOutputRepo)
	container.Register("subtaskRepo", subtaskRepo)

	taskWithPhasesQuery := query.NewTaskWithPhasesQuerySQLite(db, subtaskRepo)
	taskTimelineQuery := query.NewTaskTimelineQuerySQLite(db)
	container.Register("taskWithPhasesQuery", taskWithPhasesQuery)
	container.Register("taskTimelineQuery", taskTimelineQuery)

	// 5. Event Dispatcher and SSE Broker
	dispatcher := eventimpl.NewDispatcherMemory()
	container.Register("dispatcher", dispatcher)

	sseBroker := sse.NewBroker(dispatcher)
	container.Register("sseBroker", sseBroker)

	// 6. Harness Adapter and Prompt Builder
	apiHost := cfg.Server.Host
	if apiHost == "" {
		apiHost = "localhost"
	}
	apiBaseURL := fmt.Sprintf("http://%s:%d/api/v1", apiHost, cfg.Server.Port)

	promptBuilder := service.NewPromptBuilder(apiBaseURL)
	container.Register("promptBuilder", promptBuilder)

	phaseConfigs := harness.BuildPhaseConfigs(
		cfg.Harness.DefaultCmd,
		cfg.Harness.DefaultModel,
		cfg.Harness.DefaultMaxRetries,
		cfg.Harness.DefaultTimeoutSec,
		convertPhaseOverrides(cfg.Harness.Phases),
	)
	liveStore := livetail.NewStore()
	harnessAdapter := harness.NewAdapter(phaseConfigs, fmt.Sprintf("%d", cfg.MCP.Port), apiBaseURL, dispatcher, liveStore)
	container.Register("harnessAdapter", harnessAdapter)
	container.Register("liveStore", liveStore)

	// 7. Use Cases
	createTaskUC := usecase.NewCreateTask(taskRepo, dispatcher)
	updateTaskUC := usecase.NewUpdateTask(taskRepo, dispatcher)
	deleteTaskUC := usecase.NewDeleteTask(taskRepo, dispatcher)
	getTaskUC := usecase.NewGetTask(taskWithPhasesQuery)
	listTasksUC := usecase.NewListTasks(taskRepo, subtaskRepo)
	advancePhaseUC := usecase.NewAdvancePhase(taskRepo, phaseOutputRepo, dispatcher)
	reportProgressUC := usecase.NewReportPhaseProgress(eventLogRepo, dispatcher)
	savePhaseOutputUC := usecase.NewSavePhaseOutput(phaseOutputRepo, dispatcher)
	createSubtasksUC := usecase.NewCreateSubtasks(subtaskRepo, dispatcher)
	updateSubtaskStatusUC := usecase.NewUpdateSubtaskStatus(subtaskRepo, dispatcher)

	container.Register("createTaskUseCase", createTaskUC)
	container.Register("updateTaskUseCase", updateTaskUC)
	container.Register("deleteTaskUseCase", deleteTaskUC)
	container.Register("getTaskUseCase", getTaskUC)
	container.Register("listTasksUseCase", listTasksUC)
	container.Register("advancePhaseUseCase", advancePhaseUC)
	container.Register("reportProgressUseCase", reportProgressUC)
	container.Register("savePhaseOutputUseCase", savePhaseOutputUC)
	container.Register("createSubtasksUseCase", createSubtasksUC)
	container.Register("updateSubtaskStatusUseCase", updateSubtaskStatusUC)

	// 8. Phase Orchestrator (created before handlers so they can depend on it)
	orchestrator := service.NewPhaseOrchestrator(
		taskRepo, phaseOutputRepo, subtaskRepo, harnessAdapter, promptBuilder, dispatcher,
	)
	container.Register("orchestrator", orchestrator)

	// 9. HTTP Handlers
	healthHandler := handler.NewHealthHandler(db)
	sseHandler := handler.NewSSEHandler(sseBroker)
	taskHandler := handler.NewTaskHandler(
		createTaskUC, updateTaskUC, deleteTaskUC,
		getTaskUC, listTasksUC, advancePhaseUC,
		createSubtasksUC, updateSubtaskStatusUC, savePhaseOutputUC,
		orchestrator,
		taskTimelineQuery,
	)

	container.Register("healthHandler", healthHandler)
	container.Register("sseHandler", sseHandler)
	container.Register("taskHandler", taskHandler)
	container.Register("liveHandler", handler.NewLiveHandler(liveStore))

	// 10. Event subscriptions (Observer pattern wiring)
	dispatcher.Subscribe(event.TaskCreated, func(evt event.Event) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		task, err := taskRepo.Find(ctx, evt.TaskID)
		if err != nil {
			slog.Error("bootstrap: failed to find task for StartFlow", "taskID", evt.TaskID, "error", err)
			return
		}
		if err := orchestrator.StartFlow(ctx, task); err != nil {
			slog.Error("bootstrap: StartFlow failed", "taskID", evt.TaskID, "error", err)
		}
	})

	dispatcher.Subscribe(event.TaskDeleted, func(evt event.Event) {
		orchestrator.KillProcess(evt.TaskID)
		liveStore.Delete(evt.TaskID)
	})

	// On harness failure (non-zero exit or timeout), drive the retry policy.
	// The orchestrator tracks attempts per task/phase and either re-dispatches
	// the phase or marks the task as failed (SPEC §13.2 / §32.3).
	dispatcher.Subscribe(event.HarnessErrorOccurred, func(evt event.Event) {
		payload, _ := evt.Payload.(map[string]any)
		phaseVal, ok := payload["phase"]
		if !ok {
			slog.Error("bootstrap: harness error without phase", "taskID", evt.TaskID)
			return
		}
		phase, ok := phaseVal.(entity.Phase)
		if !ok {
			slog.Error("bootstrap: harness error phase type assertion failed", "taskID", evt.TaskID)
			return
		}
		phaseCfg, ok := phaseConfigs[phase]
		if !ok {
			slog.Error("bootstrap: no config for failed phase", "taskID", evt.TaskID, "phase", phase)
			return
		}
		// Compose the reason from the process wait error + the captured harness
		// stdout/stderr so HandleRetry can persist it on the task (SPEC §32.3).
		waitErr, _ := payload["error"].(string)
		output, _ := payload["output"].(string)
		reason := buildHarnessReason(waitErr, output)
		// Run in a detached goroutine: HandleRetry sleeps for backoff and the
		// harness dispatch enforces its own per-attempt timeout.
		go func() {
			orchestrator.HandleRetry(context.Background(), evt.TaskID, phase, phaseCfg.MaxRetries, reason)
		}()
	})

	// Subscribe to all phase completed events -> advance to the next lane.
	phaseCompletedEvents := []event.EventType{
		event.PhasePlanningCompleted,
		event.PhaseTodoCompleted,
		event.PhaseDoingCompleted,
		event.PhaseValidatingCompleted,
		event.PhaseTestingCompleted,
	}
	for _, evtType := range phaseCompletedEvents {
		evtType := evtType
		dispatcher.Subscribe(evtType, func(evt event.Event) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := orchestrator.AdvancePhase(ctx, evt.TaskID); err != nil {
				slog.Error("bootstrap: AdvancePhase failed", "taskID", evt.TaskID, "error", err)
			}
		})
	}

	return container, nil
}

func convertPhaseOverrides(phases map[entity.Phase]config.PhaseHarnessConfig) map[entity.Phase]harness.PhaseHarnessConfig {
	result := make(map[entity.Phase]harness.PhaseHarnessConfig)
	for phase, cfg := range phases {
		result[phase] = harness.PhaseHarnessConfig{
			Cmd:        cfg.Cmd,
			Model:      cfg.Model,
			MaxRetries: cfg.MaxRetries,
			TimeoutSec: cfg.TimeoutSec,
		}
	}
	return result
}

// buildHarnessReason assembles the failure reason forwarded to HandleRetry from
// the raw harness wait error and the captured stdout/stderr. The captured
// output is the part that actually explains the failure.
func buildHarnessReason(waitErr, output string) string {
	var b strings.Builder
	if waitErr = strings.TrimSpace(waitErr); waitErr != "" {
		fmt.Fprintf(&b, "harness error: %s\n", waitErr)
	}
	if output = strings.TrimSpace(output); output != "" {
		b.WriteString("--- harness output ---\n")
		b.WriteString(output)
	}
	return strings.TrimSpace(b.String())
}
