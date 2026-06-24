package bootstrap

import (
	"context"
	"fmt"
	"kanbanai/config"
	"kanbanai/internal/di"
	"kanbanai/internal/domain/event"
	"kanbanai/internal/domain/entity"
	eventimpl "kanbanai/internal/adapter/out/event"
	"kanbanai/internal/adapter/out/harness"
	"kanbanai/internal/adapter/out/persistence/query"
	"kanbanai/internal/adapter/out/persistence/repository"
	"kanbanai/internal/adapter/out/persistence/sqlite"
	"kanbanai/internal/adapter/out/sse"
	"kanbanai/internal/application/service"
	"kanbanai/internal/application/usecase"
	"kanbanai/internal/adapter/in/http/handler"
	"log/slog"
	"os"
	"time"
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

	container.Register("taskRepo", taskRepo)
	container.Register("eventLogRepo", eventLogRepo)
	container.Register("phaseOutputRepo", phaseOutputRepo)

	taskWithPhasesQuery := query.NewTaskWithPhasesQuerySQLite(db)
	taskTimelineQuery := query.NewTaskTimelineQuerySQLite(db)
	container.Register("taskWithPhasesQuery", taskWithPhasesQuery)
	container.Register("taskTimelineQuery", taskTimelineQuery)

	// 5. Event Dispatcher and SSE Broker
	dispatcher := eventimpl.NewDispatcherMemory()
	container.Register("dispatcher", dispatcher)

	sseBroker := sse.NewBroker(dispatcher)
	container.Register("sseBroker", sseBroker)

	// 6. Harness Adapter and Prompt Builder
	promptBuilder := service.NewPromptBuilder()
	container.Register("promptBuilder", promptBuilder)

	harnessConfigs := harness.BuildPhaseConfigs(
		cfg.Harness.DefaultCmd,
		cfg.Harness.DefaultModel,
		cfg.Harness.DefaultMaxRetries,
		cfg.Harness.DefaultTimeoutSec,
		convertPhaseOverrides(cfg.Harness.Phases),
	)
	harnessAdapter := harness.NewAdapter(harnessConfigs, fmt.Sprintf("%d", cfg.MCP.Port), dispatcher)
	container.Register("harnessAdapter", harnessAdapter)

	// 7. Use Cases
	createTaskUC := usecase.NewCreateTask(taskRepo, dispatcher)
	updateTaskUC := usecase.NewUpdateTask(taskRepo, dispatcher)
	deleteTaskUC := usecase.NewDeleteTask(taskRepo, dispatcher)
	getTaskUC := usecase.NewGetTask(taskWithPhasesQuery)
	listTasksUC := usecase.NewListTasks(taskRepo)
	advancePhaseUC := usecase.NewAdvancePhase(taskRepo, phaseOutputRepo, dispatcher)
	reportProgressUC := usecase.NewReportPhaseProgress(eventLogRepo, dispatcher)
	savePhaseOutputUC := usecase.NewSavePhaseOutput(phaseOutputRepo, dispatcher)

	container.Register("createTaskUseCase", createTaskUC)
	container.Register("updateTaskUseCase", updateTaskUC)
	container.Register("deleteTaskUseCase", deleteTaskUC)
	container.Register("getTaskUseCase", getTaskUC)
	container.Register("listTasksUseCase", listTasksUC)
	container.Register("advancePhaseUseCase", advancePhaseUC)
	container.Register("reportProgressUseCase", reportProgressUC)
	container.Register("savePhaseOutputUseCase", savePhaseOutputUC)

	// 8. HTTP Handlers
	healthHandler := handler.NewHealthHandler()
	sseHandler := handler.NewSSEHandler(sseBroker)
	taskHandler := handler.NewTaskHandler(
		createTaskUC, updateTaskUC, deleteTaskUC,
		getTaskUC, listTasksUC, advancePhaseUC,
		taskTimelineQuery,
	)

	container.Register("healthHandler", healthHandler)
	container.Register("sseHandler", sseHandler)
	container.Register("taskHandler", taskHandler)

	// 9. Phase Orchestrator
	orchestrator := service.NewPhaseOrchestrator(
		taskRepo, phaseOutputRepo, harnessAdapter, promptBuilder, dispatcher,
	)
	container.Register("orchestrator", orchestrator)

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
	})

	// Subscribe to all phase completed events
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
