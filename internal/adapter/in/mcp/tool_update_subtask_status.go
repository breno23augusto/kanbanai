package mcp

import (
	"context"

	"kanbanai/internal/domain/entity"
	"kanbanai/internal/application/usecase"
	"kanbanai/internal/di"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type updateSubtaskStatusArgs struct {
	TaskID    string `json:"task_id"`
	Phase     string `json:"phase"`
	SubtaskID string `json:"subtask_id"`
	Status    string `json:"status"` // pending | in_progress | completed
}

func updateSubtaskStatusToolDef() *mcp.Tool {
	return &mcp.Tool{
		Name:        "update_subtask_status",
		Description: "Updates the status of a single subtask. Call this as you finish each sub-activity (during doing/validating/testing) so the board card reflects live per-subtask progress. Use status 'in_progress' when you start a subtask and 'completed' when it is done. The subtask_id comes from the create_subtasks response or from get_task.",
		InputSchema: jsonSchema(map[string]any{
			"task_id":    stringProp("ID of the task"),
			"phase":      stringProp("Current phase the harness is running (must match the task's current phase)"),
			"subtask_id": stringProp("ID of the subtask to update"),
			"status":     stringProp("New status: pending, in_progress or completed"),
		}, []string{"task_id", "phase", "subtask_id", "status"}),
	}
}

func updateSubtaskStatusHandler(container *di.Container) mcp.ToolHandlerFor[updateSubtaskStatusArgs, any] {
	return func(ctx context.Context, request *mcp.CallToolRequest, args updateSubtaskStatusArgs) (*mcp.CallToolResult, any, error) {
		if err := authorize(ctx, container, args.TaskID, args.Phase); err != nil {
			return nil, nil, err
		}
		uc := container.MustResolve("updateSubtaskStatusUseCase").(*usecase.UpdateSubtaskStatus)
		result, err := uc.Execute(ctx, args.TaskID, args.SubtaskID, entity.SubtaskStatus(args.Status))
		return nil, result, err
	}
}