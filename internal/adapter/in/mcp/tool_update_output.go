package mcp

import (
	"context"
	"kanbanai/internal/application/dto"
	"kanbanai/internal/application/usecase"
	"kanbanai/internal/domain/entity"
	"kanbanai/internal/di"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type updateTaskOutputArgs struct {
	TaskID  string `json:"task_id"`
	Phase   string `json:"phase"`
	Output  string `json:"output"`
	Summary string `json:"summary,omitempty"`
}

func updateTaskOutputToolDef() *mcp.Tool {
	return &mcp.Tool{
		Name:        "update_task_output",
		Description: "Saves artifacts/outputs for the current phase (plan, code, test results, etc.)",
		InputSchema: jsonSchema(map[string]any{
			"task_id": stringProp("ID of the task"),
			"phase":   stringProp("Current phase"),
			"output":  stringProp("Raw output content"),
			"summary": stringProp("Human-readable summary"),
		}, []string{"task_id", "phase", "output"}),
	}
}

func updateTaskOutputHandler(container *di.Container) mcp.ToolHandlerFor[updateTaskOutputArgs, any] {
	return func(ctx context.Context, request *mcp.CallToolRequest, args updateTaskOutputArgs) (*mcp.CallToolResult, any, error) {
		savePhaseOutput := container.MustResolve("savePhaseOutputUseCase").(*usecase.SavePhaseOutput)
		input := dto.SavePhaseOutputInput{
			TaskID:  args.TaskID,
			Phase:   entity.Phase(args.Phase),
			Output:  args.Output,
			Summary: args.Summary,
		}
		result, err := savePhaseOutput.Execute(ctx, input)
		return nil, result, err
	}
}
