package mcp

import (
	"context"
	"kanbanai/internal/application/usecase"
	"kanbanai/internal/di"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type getTaskArgs struct {
	TaskID string `json:"task_id"`
}

func getTaskToolDef() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_task",
		Description: "Retrieves the current task information including phase outputs",
		InputSchema: jsonSchema(map[string]any{
			"task_id": stringProp("ID of the task to retrieve"),
		}, []string{"task_id"}),
	}
}

func getTaskHandler(container *di.Container) mcp.ToolHandlerFor[getTaskArgs, any] {
	return func(ctx context.Context, request *mcp.CallToolRequest, args getTaskArgs) (*mcp.CallToolResult, any, error) {
		getTask := container.MustResolve("getTaskUseCase").(*usecase.GetTask)
		result, err := getTask.Execute(ctx, args.TaskID)
		return nil, result, err
	}
}
