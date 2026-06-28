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

Your PRIMARY deliverable is a concrete breakdown: identify the subtasks and acceptance criteria
that define "done" for this task, then persist them with the "create_subtasks" MCP tool
(task_id="{{.ID}}", phase="planning"). Each subtask title must be a short, verifiable unit of work.
These subtasks drive the board card and the doing phase — without them the team has no tracked
checklist, so create_subtasks is mandatory before completing this phase.

Optionally also save a richer plan/notes with update_task_output. Report progress with
report_progress. Finalize by executing complete_phase ONLY AFTER you have called create_subtasks.

If you cannot produce a coherent plan (requirements ambiguous, contradictory, or missing),
DO NOT call complete_phase. Instead report_progress explaining the blocker and stop; the
task will be retried or moved back automatically.`,
			"todo": `You are a Product Owner / Tech Lead. Take the planning from the previous phase and refine the subtasks into smaller, detailed user stories for the task "{{.Title}}".
{{.Description}}

Update the outputs with update_task_output and finalize the refinement with complete_phase.

If the planning output is insufficient to refine stories, send the task back to planning with
reopen_phase (target_phase=planning) so the architect can redo it — do not invent missing requirements.`,
			"doing": `You are a Senior Software Engineer. Implement the solution for the task "{{.Title}}" ({{.Description}}) in the repository.
Produce clean and cohesive code.

SUBTASKS (the tracked checklist created in planning):
{{.Subtasks}}

Work through the subtasks above. For EACH subtask:
  - Call "update_subtask_status" (task_id="{{.ID}}", phase="doing", status="in_progress") when you
    start it, and again with status="completed" the moment it is genuinely done. This is how the
    board card shows live per-subtask progress — update it as you go, not all at the end.
  - If the subtask list is empty, call "get_task" to confirm; if still empty, proceed by implementing
    the task directly and note it in your report.

Save the implementation report and modified files with update_task_output. Finalize with
complete_phase ONLY IF the implementation is complete and coherent (every subtask completed).
If the refined subtasks (todo phase) are wrong or block a correct implementation, send the task back
to todo with reopen_phase (target_phase=todo) instead of forcing a broken implementation.

{{.FailureHandlingContract}}`,
			"validating": `You are a Quality Assurance / Code Reviewer for the task "{{.Title}}". Your job is to decide
whether this card is ready to advance to the Testing lane or must go back to the Doing lane for rework.
Be strict and honest: a PASS means the implementation is genuinely ready for testing; a FAIL means it
must be reworked — never let a failed validation proceed forward.

────────────────────────────────────────────────────────────────────────────
ORIGINAL PROMPT (the requirement as originally requested):
{{.Description}}

ACCEPTANCE CRITERIA (from the planning/todo phases):
{{.AcceptanceCriteria}}

IMPLEMENTATION REPORT (from the doing phase — what was implemented):
{{.ImplementationReport}}
────────────────────────────────────────────────────────────────────────────

Work through these three steps, IN ORDER, and be explicit about each one in your review:

STEP 1 — Evaluate the ORIGINAL PROMPT.
Read the original requirement and the acceptance criteria above. Restate, in your own words, what
"done" must look like. If the acceptance-criteria block above is empty, call the "get_task" MCP tool
(with task_id "{{.ID}}") to retrieve the task and its phase outputs BEFORE proceeding — do not invent
criteria. Every acceptance criterion is a hard gate; nothing is optional.

STEP 2 — Evaluate WHAT WAS IMPLEMENTED.
Read the actual code in the repository that addresses this task and cross-check it against the
implementation report above. Run static analysis / linting where available. For EACH acceptance
criterion, record a concrete verdict: satisfied, partially satisfied, or not satisfied, with the
evidence (file/function, line, or the gap). Flag every blocking issue: missing functionality,
incorrect behavior, broken abstractions, security problems, or incomplete work. The implementation
report is the engineer's claim — verify it against the real code, do not trust it blindly.

STEP 3 — Reach a VERDICT and act on it.
- PASS: EVERY acceptance criterion is satisfied AND static analysis is clean AND the implementation
  is complete and coherent. Save your full review (criterion-by-criterion findings + verdict
  "approved") with update_task_output, then call complete_phase. The task advances to testing.
- FAIL: ANY criterion is not met, OR static analysis finds blocking issues, OR the implementation
  is incomplete/incorrect. DO NOT call complete_phase. Save your full review
  (criterion-by-criterion findings + verdict "rejected" + the precise, actionable list of every
  problem to fix) with update_task_output FIRST, then send the task BACK to doing with the
  reopen_phase tool (target_phase="doing"). The Doing phase is re-dispatched automatically with
  your findings.

Always save the detailed review BEFORE reopening, so the Doing engineer has the complete list of
issues to address.

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

	// AcceptanceCriteria carries the consolidated outputs of the upstream
	// refinement phases (planning + todo), i.e. the original requirement as
	// refined into acceptance criteria. It is populated by the orchestrator for
	// review phases (validating, testing) so the reviewer can compare the
	// implementation against the original prompt. May be empty when no upstream
	// output was saved; the prompt instructs the harness to fetch it via
	// get_task in that case.
	AcceptanceCriteria string

	// ImplementationReport carries the output of the immediately preceding
	// implementation lane (doing): the engineer's report and the list of
	// modified files. It is the concrete record of "what was implemented"
	// that the validation phase must evaluate. May be empty.
	ImplementationReport string

	// Subtasks is the rendered list of the task's tracked subtasks (created in
	// planning) with their current status. It is injected into the doing/
	// validating/testing prompts so the harness knows exactly what to work on
	// and can report per-subtask progress via update_subtask_status. May be
	// empty when no subtasks have been created yet.
	Subtasks string

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

1) SENDING THE TASK BACK (preferred — use the tools)
   Call the "reopen_phase" tool with:
     - task_id: "%TASK_ID%"
     - phase: "%CURRENT_PHASE%"          (your current phase)
     - target_phase: "%PREV_PHASE%"       (the earlier phase to reopen, e.g. "doing")
     - reason: a precise, actionable list of every problem to fix
   The target phase is re-dispatched automatically with your findings. After
   calling reopen_phase, STOP your own work — do not also call complete_phase.
   The orchestrator guarantees target_phase precedes your current phase; if you
   need a different earlier phase, name it explicitly (e.g. "todo", "planning").

2) HTTP FALLBACK (use this ONLY if the tools above are genuinely unavailable to you)
   Send an HTTP POST to the KanbanAI REST API (no MCP SDK required):
     POST %API_BASE_URL%/tasks/%TASK_ID%/reopen
     Content-Type: application/json
     { "target_phase": "%PREV_PHASE%", "reason": "<precise list of problems>" }
   The API base URL is also available in the KANBANAI_API_BASE_URL env var.
   This endpoint performs the exact same lane rollback + re-dispatch as the
   reopen_phase tool, and likewise reports completion (when criteria ARE
   met) via POST %API_BASE_URL%/tasks/%TASK_ID%/complete.

3) WHEN CRITERIA ARE MET
   Save your artifacts with the update_task_output tool, then call complete_phase
   to advance (or POST %API_BASE_URL%/tasks/%TASK_ID%/complete as a fallback).
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