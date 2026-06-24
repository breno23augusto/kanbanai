package mcp

import (
	"context"
	"kanbanai/internal/application/usecase"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/di"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type reportProgressArgs struct {
	TaskID  string `json:"task_id"`
	Phase   string `json:"phase"`
	Message string `json:"message"`
}

func reportProgressToolDef() *mcp.Tool {
	return &mcp.Tool{
		Name:        "report_progress",
		Description: "Reports partial progress for the current phase",
		InputSchema: jsonSchema(map[string]any{
			"task_id": stringProp("ID of the task"),
			"phase":   stringProp("Current phase"),
			"message": stringProp("Progress message"),
		}, []string{"task_id", "phase", "message"}),
	}
}

func reportProgressHandler(container *di.Container) mcp.ToolHandlerFor[reportProgressArgs, any] {
	return func(ctx context.Context, request *mcp.CallToolRequest, args reportProgressArgs) (*mcp.CallToolResult, any, error) {
		if err := authorize(ctx, container, args.TaskID, args.Phase); err != nil {
			return nil, nil, err
		}
		reportProgress := container.MustResolve("reportProgressUseCase").(*usecase.ReportPhaseProgress)
		err := reportProgress.Execute(ctx, args.TaskID, entity.Phase(args.Phase), args.Message)
		return nil, map[string]any{"status": "ok"}, err
	}
}