package harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type CommandBuilder struct {
	mcpPort string
}

func NewCommandBuilder(mcpPort string) *CommandBuilder {
	return &CommandBuilder{mcpPort: mcpPort}
}

func (b *CommandBuilder) Build(ctx context.Context, harnessCmd string, modelName string, taskID string, prompt string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, harnessCmd, "--model", modelName, "--prompt", prompt)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("KANBANAI_TASK_ID=%s", taskID),
		fmt.Sprintf("KANBANAI_MCP_PORT=%s", b.mcpPort),
		fmt.Sprintf("KANBANAI_MCP_URL=http://localhost:%s/mcp/sse", b.mcpPort),
	)
	return cmd, nil
}
