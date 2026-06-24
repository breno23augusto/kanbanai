package service

import (
	"bytes"
	"strings"
	"text/template"

	"kanbanai/internal/domain/entity"
)

// PromptBuilder renders the system prompt dispatched to the harness for each
// phase. Every prompt carries the same failure-handling contract (see
// failureHandlingContract): a phase that detects problems which require
// rework in an EARLIER phase must send the task back with reopen_phase (or the
// HTTP equivalent), and must NEVER call complete_phase while acceptance
// criteria are unmet. This is what prevents a failed validation from being
// carried forward to testing/done (SPEC §6.3.7).
type PromptBuilder struct {
	templates  map[string]string
	apiBaseURL string
}

func NewPromptBuilder(apiBaseURL string) *PromptBuilder {
	return &PromptBuilder{
		apiBaseURL: strings.TrimRight(apiBaseURL, "/"),
		templates: map[string]string{
			"planning": `You are a software architect. Analyze the requirements for the task "{{.Title}}".
{{.Description}}

Identify and save subtasks and acceptance criteria using the update_task_output MCP tool.
Report progress with report_progress. Finalize by executing complete_phase.

If you cannot produce a coherent plan (requirements ambiguous, contradictory, or missing),
DO NOT call complete_phase. Instead report_progress explaining the blocker and stop; the
task will be retried or moved back automatically.`,
			"todo": `You are a Product Owner / Tech Lead. Take the planning from the previous phase and refine the subtasks into smaller, detailed user stories for the task "{{.Title}}".
{{.Description}}

Update the outputs with update_task_output and finalize the refinement with complete_phase.

If the planning output is insufficient to refine stories, send the task back to planning with
reopen_phase (target_phase=planning) so the architect can redo it — do not invent missing requirements.`,
			"doing": `You are a Senior Software Engineer. Implement the solution for the task "{{.Title}}" ({{.Description}}) in the repository.
Produce clean and cohesive code. Save the implementation report and modified files with update_task_output.

When you finish, finalize with complete_phase ONLY IF the implementation is complete and coherent.
If the refined subtasks (todo phase) are wrong or block a correct implementation, send the task back
to todo with reopen_phase (target_phase=todo) instead of forcing a broken implementation.

{{.FailureHandlingContract}}`,
			"validating": `You are a Quality Assurance / Reviewer. Analyze the code produced in the Doing phase for the task "{{.Title}}" ({{.Description}}).

Perform static analysis and validate that ALL acceptance criteria have been met. Be strict and honest:
a pass here means the implementation is ready for the Testing phase; a fail means it must be reworked.

DECISION RULE (mandatory):
- If EVERY acceptance criterion is satisfied and static analysis is clean: save your review with
  update_task_output and finalize with complete_phase. The task advances to testing.
- If ANY criterion is NOT met, OR static analysis finds blocking issues, OR the implementation is
  incomplete/incorrect: DO NOT call complete_phase. Instead send the task BACK to doing with the
  reopen_phase tool (target_phase="doing") and a clear reason listing every problem to fix. The
  Doing phase will be re-dispatched automatically with your findings. Never let a failed validation
  proceed forward — that defeats the purpose of this phase.

Save the detailed review (findings + verdict) with update_task_output BEFORE reopening, so the Doing
engineer has the full list of issues to address.

{{.FailureHandlingContract}}`,
			"testing": `You are a Software Test Engineer. Write and execute automated unit/integration tests for the task "{{.Title}}" to cover the code implemented in the Doing phase.

Run the tests and capture real results. Save the test code and results with update_task_output.

DECISION RULE (mandatory):
- If tests pass and adequately cover the acceptance criteria: finalize with complete_phase. The task
  reaches Done.
- If tests FAIL, or coverage is insufficient for the acceptance criteria: DO NOT call complete_phase.
  Send the task back to doing with reopen_phase (target_phase="doing") and a reason listing the failing
  tests / missing coverage. Do not mark the task Done with failing tests.

{{.FailureHandlingContract}}`,
		},
	}
}

type PromptData struct {
	Title        string
	Description  string
	ID           string
	Phase        string
	MCPServerURL string
	APIBaseURL   string

	// FailureHandlingContract is the rendered, phase-agnostic instructions that
	// teach the harness how to send a task back and how to fall back to HTTP
	// when it has no MCP client. It is injected automatically by Build.
	FailureHandlingContract string
}

func (b *PromptBuilder) Build(phase string, data PromptData) (string, error) {
	tmplStr, ok := b.templates[phase]
	if !ok {
		tmplStr = "Execute the {{.Phase}} phase for task \"{{.Title}}\". Use MCP tools to report progress and complete the phase.\n\n{{.FailureHandlingContract}}"
	}

	if data.APIBaseURL == "" {
		data.APIBaseURL = b.apiBaseURL
	}
	data.FailureHandlingContract = b.failureHandlingContract(data.APIBaseURL, data.ID, phase)

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

// failureHandlingContract is the shared block appended to the implementation,
// validation and testing prompts. It tells the harness, in tool-agnostic
// terms, how to (a) send the task back to an earlier phase when it detects
// problems, and (b) do the same over plain HTTP when it has no MCP client
// (e.g. pi). Keeping it in one place guarantees every phase that can detect
// failures instructs the harness consistently (SPEC §6.3.7).
//
// Placeholders between percent signs (%TASK_ID%, %CURRENT_PHASE%, %PREV_PHASE%,
// %API_BASE_URL%) are substituted with the concrete values AFTER the block is
// injected, because the contract is passed to the template as a literal string
// value (FailureHandlingContract) and is not itself template-parsed.
func (b *PromptBuilder) failureHandlingContract(apiBaseURL, taskID, currentPhase string) string {
	prev := previousPhaseForReopen(currentPhase)
	s := `FAILURE-HANDLING CONTRACT (read carefully — this is mandatory):

You are part of an automated Kanban flow. Phases advance ONLY when their exit
criteria are fully met. When you detect that an EARLIER phase produced work
that is incorrect or incomplete, you MUST send the task back to that phase for
rework — never call complete_phase to push a known failure forward.

1) SENDING THE TASK BACK (preferred — use MCP tools)
   Call the "reopen_phase" MCP tool with:
     - task_id: "%TASK_ID%"
     - phase: "%CURRENT_PHASE%"          (your current phase)
     - target_phase: "%PREV_PHASE%"       (the earlier phase to reopen, e.g. "doing")
     - reason: a precise, actionable list of every problem to fix
   The target phase is re-dispatched automatically with your findings. After
   calling reopen_phase, STOP your own work — do not also call complete_phase.
   The orchestrator guarantees target_phase precedes your current phase; if you
   need a different earlier phase, name it explicitly (e.g. "todo", "planning").

2) HTTP FALLBACK (use this ONLY if your harness has no MCP client, e.g. pi)
   Send an HTTP POST to the KanbanAI REST API (no MCP SDK required):
     POST %API_BASE_URL%/tasks/%TASK_ID%/reopen
     Content-Type: application/json
     { "target_phase": "%PREV_PHASE%", "reason": "<precise list of problems>" }
   The API base URL is also available in the KANBANAI_API_BASE_URL env var.
   This endpoint performs the exact same lane rollback + re-dispatch as the
   reopen_phase MCP tool, and likewise reports completion (when criteria ARE
   met) via POST %API_BASE_URL%/tasks/%TASK_ID%/complete.

3) WHEN CRITERIA ARE MET
   Save your artifacts with update_task_output (MCP) or just proceed, then call
   complete_phase (MCP) or POST %API_BASE_URL%/tasks/%TASK_ID%/complete.
   Use complete ONLY for a genuine pass; use reopen for any failure.
`
	repl := func(old, new string) { s = strings.ReplaceAll(s, old, new) }
	repl("%TASK_ID%", taskID)
	repl("%CURRENT_PHASE%", currentPhase)
	repl("%PREV_PHASE%", string(prev))
	repl("%API_BASE_URL%", apiBaseURL)
	return s
}

// previousPhaseForReopen returns the phase the harness should typically reopen
// to from currentPhase: the immediately preceding lane (validating -> doing,
// testing -> doing, doing -> todo, todo -> planning). For planning it returns
// planning itself as a safe no-op default (the contract still names it).
func previousPhaseForReopen(currentPhase string) entity.Phase {
	return previousPhaseForReopenPhase(entity.Phase(currentPhase))
}

func previousPhaseForReopenPhase(p entity.Phase) entity.Phase {
	if prev, ok := p.Prev(); ok {
		return prev
	}
	return p
}