package mcp

import (
	"context"
	"kanbanai/internal/application/usecase"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/di"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type completePhaseArgs struct {
	TaskID  string `json:"task_id"`
	Phase   string `json:"phase"`
	Summary string `json:"summary"`
}

func completePhaseToolDef() *mcp.Tool {
	return &mcp.Tool{
		Name:        "complete_phase",
		Description: "Marks the current phase as completed. The next phase is started automatically by the orchestrator.",
		InputSchema: jsonSchema(map[string]any{
			"task_id": stringProp("ID of the task"),
			"phase":   stringProp("Current phase being completed"),
			"summary": stringProp("Summary of what was accomplished"),
		}, []string{"task_id", "phase", "summary"}),
	}
}

func completePhaseHandler(container *di.Container) mcp.ToolHandlerFor[completePhaseArgs, any] {
	return func(ctx context.Context, request *mcp.CallToolRequest, args completePhaseArgs) (*mcp.CallToolResult, any, error) {
		advancePhase := container.MustResolve("advancePhaseUseCase").(*usecase.AdvancePhase)
		err := advancePhase.Execute(ctx, args.TaskID, entity.Phase(args.Phase), args.Summary)
		return nil, map[string]any{"status": "completed"}, err
	}
}
