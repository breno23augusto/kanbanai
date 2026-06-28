package mcp

import (
	"encoding/json"
	"kanbanai/internal/di"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterTools(server *mcp.Server, container *di.Container) {
	mcp.AddTool(server, reportProgressToolDef(), reportProgressHandler(container))
	mcp.AddTool(server, updateTaskOutputToolDef(), updateTaskOutputHandler(container))
	mcp.AddTool(server, completePhaseToolDef(), completePhaseHandler(container))
	mcp.AddTool(server, reopenPhaseToolDef(), reopenPhaseHandler(container))
	mcp.AddTool(server, getTaskToolDef(), getTaskHandler(container))
	mcp.AddTool(server, createSubtasksToolDef(), createSubtasksHandler(container))
	mcp.AddTool(server, updateSubtaskStatusToolDef(), updateSubtaskStatusHandler(container))
}

func jsonSchema(props map[string]any, required []string) json.RawMessage {
	schema := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	data, _ := json.Marshal(schema)
	return data
}

func stringProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}
