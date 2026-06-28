package mcp

import (
	"context"

	"kanbanai/internal/application/dto"
	"kanbanai/internal/application/usecase"
	"kanbanai/internal/di"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type subtaskItemArg struct {
	Title string `json:"title"`
}

type createSubtasksArgs struct {
	TaskID   string            `json:"task_id"`
	Phase   string            `json:"phase"`
	Subtasks []subtaskItemArg `json:"subtasks"`
}

func createSubtasksToolDef() *mcp.Tool {
	return &mcp.Tool{
		Name:        "create_subtasks",
		Description: "Replaces the task's subtask list with the provided items. Call this during the planning phase to persist the breakdown/acceptance criteria you identified, so they show up on the board card and the detail drawer. Each item becomes a tracked subtask with status 'pending'. Returns the created subtasks (with their ids) so later phases can update them by id.",
		InputSchema: jsonSchema(map[string]any{
			"task_id":  stringProp("ID of the task"),
			"phase":    stringProp("Current phase the harness is running (must match the task's current phase, e.g. planning)"),
			"subtasks": map[string]any{
				"type":        "array",
				"description": "Ordered list of subtasks to create (replaces any existing ones)",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"title": stringProp("Short title of the subtask / acceptance criterion"),
					},
					"required": []string{"title"},
				},
			},
		}, []string{"task_id", "phase", "subtasks"}),
	}
}

func createSubtasksHandler(container *di.Container) mcp.ToolHandlerFor[createSubtasksArgs, any] {
	return func(ctx context.Context, request *mcp.CallToolRequest, args createSubtasksArgs) (*mcp.CallToolResult, any, error) {
		if err := authorize(ctx, container, args.TaskID, args.Phase); err != nil {
			return nil, nil, err
		}
		items := make([]dto.SubtaskInput, 0, len(args.Subtasks))
		for _, it := range args.Subtasks {
			items = append(items, dto.SubtaskInput{Title: it.Title})
		}
		uc := container.MustResolve("createSubtasksUseCase").(*usecase.CreateSubtasks)
		result, err := uc.Execute(ctx, args.TaskID, items)
		return nil, result, err
	}
}