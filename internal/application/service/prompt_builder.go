package service

import (
	"bytes"
	"text/template"
)

type PromptBuilder struct {
	templates map[string]string
}

func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{
		templates: map[string]string{
			"planning": `You are a software architect. Analyze the requirements for the task "{{.Title}}".
{{.Description}}
Identify and save subtasks and acceptance criteria using update_task_output.
Report progress with report_progress.
Finalize by executing complete_phase.`,
			"todo": `You are a Product Owner / Tech Lead. Take the planning from the previous phase and refine the subtasks into smaller, detailed user stories for the task "{{.Title}}".
{{.Description}}
Update the outputs and finalize the refinement.`,
			"doing": `You are a Senior Software Engineer. Implement the solution for the task "{{.Title}}" ({{.Description}}) in the repository.
Produce clean and cohesive code. Save the implementation report and modified files.`,
			"validating": `You are a Quality Assurance / Reviewer. Analyze the code produced in the previous phase for the task "{{.Title}}".
Perform static analysis and validate that all acceptance criteria have been met.`,
			"testing": `You are a Software Test Engineer. Write and execute automated unit/integration tests for the task "{{.Title}}" to cover the code implemented in the Doing phase.`,
		},
	}
}

type PromptData struct {
	Title        string
	Description  string
	ID           string
	Phase        string
	MCPServerURL string
}

func (b *PromptBuilder) Build(phase string, data PromptData) (string, error) {
	tmplStr, ok := b.templates[phase]
	if !ok {
		tmplStr = "Execute the {{.Phase}} phase for task \"{{.Title}}\". Use MCP tools to report progress and complete the phase."
	}

	tmpl, err := template.New("prompt").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}