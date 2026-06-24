package mcp

import (
	"context"
	"fmt"

	"kanbanai/internal/domain/entity"
	"kanbanai/internal/domain/port"
	"kanbanai/internal/di"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// reopenPhaseArgs is the payload for the reopen_phase MCP tool. The harness
// calls it from a downstream phase (typically Validating) when its review
// detected problems that must be reworked in an earlier lane (typically Doing),
// instead of letting the task advance with unresolved failures (SPEC §6.3.7).
type reopenPhaseArgs struct {
	TaskID       string `json:"task_id"`
	Phase        string `json:"phase"`         // current phase of the harness (e.g. "validating")
	TargetPhase  string `json:"target_phase"`  // earlier phase to reopen (e.g. "doing")
	Reason       string `json:"reason"`        // why validation failed / what must be fixed
}

func reopenPhaseToolDef() *mcp.Tool {
	return &mcp.Tool{
		Name:        "reopen_phase",
		Description: "Moves the task BACK to an earlier phase (e.g. from validating back to doing) and re-dispatches it, so problems found during validation/review get reworked instead of carried forward. Use this when the current phase detected failures that require changes in a previous phase. After calling it, stop your own work — the target phase is dispatched automatically.",
		InputSchema: jsonSchema(map[string]any{
			"task_id":      stringProp("ID of the task"),
			"phase":        stringProp("Current phase the harness is running (must match the task's current phase)"),
			"target_phase": stringProp("Earlier phase to reopen (e.g. doing). Must precede the current phase."),
			"reason":       stringProp("Description of the failures/problems found that justify reopening"),
		}, []string{"task_id", "phase", "target_phase", "reason"}),
	}
}

func reopenPhaseHandler(container *di.Container) mcp.ToolHandlerFor[reopenPhaseArgs, any] {
	return func(ctx context.Context, request *mcp.CallToolRequest, args reopenPhaseArgs) (*mcp.CallToolResult, any, error) {
		// authorize against the CURRENT phase: the harness is still running in
		// it (e.g. validating) when it asks to reopen an earlier lane.
		if err := authorize(ctx, container, args.TaskID, args.Phase); err != nil {
			return nil, nil, err
		}
		target := entity.Phase(args.TargetPhase)
		if target == "" {
			return nil, nil, fmt.Errorf("target_phase is required")
		}

		orchestrator, ok := container.Resolve("orchestrator").(port.PhaseOrchestratorPort)
		if !ok {
			return nil, nil, fmt.Errorf("orchestrator not available")
		}

		if err := orchestrator.ReopenPhase(ctx, args.TaskID, target, args.Reason); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"status":        "reopened",
			"target_phase":  string(target),
			"message":       "task moved back and target phase dispatched; stop your current work",
		}, nil
	}
}