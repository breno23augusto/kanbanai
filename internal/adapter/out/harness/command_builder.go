package harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type CommandBuilder struct {
	mcpPort    string
	apiBaseURL string
}

func NewCommandBuilder(mcpPort, apiBaseURL string) *CommandBuilder {
	return &CommandBuilder{mcpPort: mcpPort, apiBaseURL: apiBaseURL}
}

func (b *CommandBuilder) Build(ctx context.Context, harnessCmd string, modelName string, taskID string, phase string, workspace string, prompt string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, harnessCmd, "--model", modelName, "--prompt", prompt)
	env := append(os.Environ(),
		fmt.Sprintf("KANBANAI_TASK_ID=%s", taskID),
		fmt.Sprintf("KANBANAI_PHASE=%s", phase),
		fmt.Sprintf("KANBANAI_MCP_PORT=%s", b.mcpPort),
		fmt.Sprintf("KANBANAI_MCP_URL=http://localhost:%s/mcp/sse", b.mcpPort),
		fmt.Sprintf("KANBANAI_API_BASE_URL=%s", b.apiBaseURL),
	)
	// Per-task workspace overrides the server's configured default (PI_HARNESS_CWD).
	if workspace != "" {
		env = append(env, fmt.Sprintf("PI_HARNESS_CWD=%s", workspace))
	}
	cmd.Env = env
	return cmd, nil
}
