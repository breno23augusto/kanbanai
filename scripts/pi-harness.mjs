// pi-harness.mjs — KanbanAI harness implemented with the pi SDK.
import { writeSync } from "node:fs"; // synchronous writes to fd 2 so the live tail (livetail.Store) sees agent output in real time; Node's process.stderr buffers async and would delay the stream.
//
// pi has no built-in MCP client, so the KanbanAI phase operations (create_subtasks,
// update_subtask_status, update_task_output, get_task, reopen_phase, complete_phase)
// are exposed to the agent as pi customTools with the SAME names as the MCP tools.
// Each customTool is a thin wrapper over KanbanAI's REST API, which invokes the
// exact same use cases the MCP server does. This lets the prompt (which is written
// in tool-agnostic terms: "call create_subtasks", "call complete_phase") drive a
// pi-based harness identically to an MCP-capable one.
//
// Persistence guarantee: the full assistant output is accumulated and, if the agent
// did not already persist it via update_task_output, the harness saves it after the
// run (so a phase is never left with only a truncated tail). Phase finalization is
// the agent's responsibility (complete_phase / reopen_phase); if the agent exits
// without finalizing, the harness falls back to auto-completing with a short summary
// so the lane never hangs.
//
// Invoked by scripts/pi-harness.sh as: node pi-harness.mjs --model <m> --prompt <p>
// Env: KANBANAI_TASK_ID, KANBANAI_PHASE, KANBANAI_API_URL, PI_PKG_DIR,
//      PI_HARNESS_MODEL, PI_HARNESS_TOOLS (csv, optional), PI_HARNESS_CWD (optional).

const pkgDir = process.env.PI_PKG_DIR;
if (!pkgDir) {
  console.error("pi-harness: PI_PKG_DIR not set");
  process.exit(2);
}

const sdk = await import(`${pkgDir}/dist/index.js`);
const { AuthStorage, ModelRegistry, SessionManager, createAgentSession, defineTool } = sdk;
const { Type } = await import(`${pkgDir}/node_modules/typebox/build/index.mjs`);

// --- parse args -------------------------------------------------------------
let modelPattern = process.env.PI_HARNESS_MODEL || "ollama/deepseek-v4-flash:cloud";
let prompt = "";
for (let i = 2; i < process.argv.length; i++) {
  const a = process.argv[i];
  if (a === "--model") { modelPattern = process.argv[++i]; continue; }
  if (a === "--prompt") { prompt = process.argv[++i]; continue; }
}

if (!prompt) {
  console.error("pi-harness: --prompt is required");
  process.exit(2);
}

const taskId = process.env.KANBANAI_TASK_ID;
const phase = process.env.KANBANAI_PHASE || "";
const apiUrl = (process.env.KANBANAI_API_URL || process.env.KANBANAI_API_BASE_URL || "http://localhost:8080").replace(/\/$/, "");
if (!taskId) {
  console.error("pi-harness: KANBANAI_TASK_ID not set");
  process.exit(2);
}

// --- KanbanAI REST bridge ---------------------------------------------------
// Each tool mirrors its MCP counterpart (same name + semantics) over the REST API.
let phaseFinalized = null;   // null | "complete" | "reopen"
let outputSaved = false;     // did the agent call update_task_output itself?

async function api(method, path, body) {
  const res = await fetch(`${apiUrl}/api/v1${path}`, {
    method,
    headers: { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await res.text().catch(() => "");
  if (!res.ok) {
    throw new Error(`${method} ${path} failed: ${res.status} ${res.statusText} ${text}`);
  }
  return text ? JSON.parse(text) : {};
}

const ok = (msg) => ({ content: [{ type: "text", text: msg }], details: {} });
const err = (e) => ({ content: [{ type: "text", text: `Error: ${e?.message || e}` }], details: { error: String(e) } });

const kanbanTools = [
  defineTool({
    name: "get_task",
    label: "Get Task",
    description: "Fetch the full task detail (description, current phase, status, phase outputs, and subtasks) for the active KanbanAI task. Use this to recover context — e.g. acceptance criteria, the implementation report, or the subtask checklist — when the prompt blocks are empty.",
    parameters: Type.Object({
      task_id: Type.String({ description: "ID of the task" }),
    }),
    execute: async (_id, params) => {
      try {
        const r = await api("GET", `/tasks/${params.task_id}`);
        return ok(JSON.stringify(r.data ?? r, null, 2));
      } catch (e) { return err(e); }
    },
  }),
  defineTool({
    name: "create_subtasks",
    label: "Create Subtasks",
    description: "Replace the task's subtask checklist (planning phase). Provide an array of subtask titles; each must be a short, verifiable unit of work. Any existing subtasks are deleted first. Mandatory in planning before complete_phase.",
    parameters: Type.Object({
      task_id: Type.String({ description: "ID of the task" }),
      subtasks: Type.Array(Type.Object({
        title: Type.String({ description: "Short, verifiable subtask title" }),
      }), { description: "Subtasks to create (replaces existing)" }),
    }),
    execute: async (_id, params) => {
      try {
        const r = await api("POST", `/tasks/${params.task_id}/subtasks`, { subtasks: params.subtasks });
        return ok(`Created ${params.subtasks.length} subtask(s):\n${JSON.stringify(r.data ?? r, null, 2)}`);
      } catch (e) { return err(e); }
    },
  }),
  defineTool({
    name: "update_subtask_status",
    label: "Update Subtask Status",
    description: "Advance a single subtask's status (doing phase). Set status to in_progress when you start it, and completed the moment it is genuinely done. Update as you go so the board card shows live progress.",
    parameters: Type.Object({
      task_id: Type.String({ description: "ID of the task" }),
      subtask_id: Type.String({ description: "ID of the subtask (from create_subtasks / get_task)" }),
      status: Type.String({ description: "New status: pending | in_progress | completed" }),
    }),
    execute: async (_id, params) => {
      try {
        const r = await api("PATCH", `/tasks/${params.task_id}/subtasks/${params.subtask_id}`, { status: params.status });
        return ok(`Subtask ${params.subtask_id} -> ${params.status}\n${JSON.stringify(r.data ?? r, null, 2)}`);
      } catch (e) { return err(e); }
    },
  }),
  defineTool({
    name: "update_task_output",
    label: "Save Phase Output",
    description: "Persist the raw artifacts/output and/or a human-readable summary for the current phase (plan, implementation report, test results, review). Call this to store your work product so later phases can read it. phase defaults to the task's current phase.",
    parameters: Type.Object({
      task_id: Type.String({ description: "ID of the task" }),
      phase: Type.String({ description: "Phase this output belongs to (e.g. planning, doing, validating). Defaults to the current phase." }),
      output: Type.String({ description: "Raw output content (full text / artifacts)" }),
      summary: Type.Optional(Type.String({ description: "Human-readable summary (optional)" })),
    }),
    execute: async (_id, params) => {
      try {
        const r = await api("PUT", `/tasks/${params.task_id}/output`, {
          phase: params.phase || phase,
          output: params.output,
          summary: params.summary || "",
        });
        outputSaved = true;
        return ok(`Saved phase output (${params.output.length} chars) for phase ${params.phase || phase}.\n${JSON.stringify(r.data ?? r, null, 2)}`);
      } catch (e) { return err(e); }
    },
  }),
  defineTool({
    name: "reopen_phase",
    label: "Reopen Phase",
    description: "Send the task BACK to an earlier phase for rework (when you detect problems). The target phase is re-dispatched automatically with your reason. After calling this, STOP — do not also call complete_phase. target_phase must precede your current phase (e.g. doing, todo, planning).",
    parameters: Type.Object({
      task_id: Type.String({ description: "ID of the task" }),
      target_phase: Type.String({ description: "Earlier phase to reopen (e.g. doing, todo, planning)" }),
      reason: Type.String({ description: "Precise, actionable list of every problem to fix" }),
    }),
    execute: async (_id, params) => {
      try {
        const r = await api("POST", `/tasks/${params.task_id}/reopen`, {
          target_phase: params.target_phase,
          reason: params.reason,
        });
        phaseFinalized = "reopen";
        return ok(`Task sent back to ${params.target_phase}. Stop your work now — do not call complete_phase.\n${JSON.stringify(r.data ?? r, null, 2)}`);
      } catch (e) { return err(e); }
    },
  }),
  defineTool({
    name: "complete_phase",
    label: "Complete Phase",
    description: "Advance the task to the NEXT phase (your current phase is done). Only call this when exit criteria are genuinely met. For planning you MUST have called create_subtasks first. Optionally pass a short summary.",
    parameters: Type.Object({
      task_id: Type.String({ description: "ID of the task" }),
      phase: Type.Optional(Type.String({ description: "Current phase (defaults to the task's current phase)" })),
      summary: Type.Optional(Type.String({ description: "Short summary of what was accomplished" })),
    }),
    execute: async (_id, params) => {
      try {
        const r = await api("POST", `/tasks/${params.task_id}/complete`, {
          phase: params.phase || phase,
          summary: params.summary || "",
        });
        phaseFinalized = "complete";
        return ok(`Phase completed — task advanced to the next lane.\n${JSON.stringify(r.data ?? r, null, 2)}`);
      } catch (e) { return err(e); }
    },
  }),
];

const kanbanToolNames = kanbanTools.map((t) => t.name);

// --- resolve model ----------------------------------------------------------
const authStorage = AuthStorage.create();
const modelRegistry = ModelRegistry.create(authStorage);

function resolveModel(pattern) {
  const slash = pattern.indexOf("/");
  let provider = slash >= 0 ? pattern.slice(0, slash) : pattern;
  let id = slash >= 0 ? pattern.slice(slash + 1) : "";
  let model = id ? modelRegistry.find(provider, id) : null;
  if (model) return model;
  return null; // resolved below via getAvailable
}

let model = resolveModel(modelPattern);
if (!model) {
  const available = await modelRegistry.getAvailable();
  const lower = modelPattern.toLowerCase();
  model = available.find((m) => `${m.provider}/${m.id}`.toLowerCase().includes(lower))
       || available.find((m) => m.provider.toLowerCase() === lower)
       || available[0];
}
if (!model) {
  console.error(`pi-harness: no model available for pattern "${modelPattern}"`);
  process.exit(2);
}

console.error(`pi-harness: task=${taskId} phase=${phase} model=${model.provider}/${model.id}`);

// --- run agent --------------------------------------------------------------
// The prompt (from PromptBuilder) already instructs the agent which tools to
// call and when to complete/reopen. We only add a short framing reminder that
// the KanbanAI lifecycle tools are available and that the agent owns phase
// finalization (complete to advance, reopen to send back).
const framedPrompt =
  `${prompt}\n\n` +
  `---\nYou are running as an automated KanbanAI phase worker. The KanbanAI lifecycle ` +
  `tools are available to you as native tools (get_task, create_subtasks, ` +
  `update_subtask_status, update_task_output, reopen_phase, complete_phase). Use ` +
  `update_task_output to persist your full work product, and finalize the phase ` +
  `yourself: call complete_phase when the exit criteria are met, or reopen_phase to ` +
  `send the task back for rework. Do not just stop without finalizing unless you are ` +
  `blocked — if blocked, call report_progress and explain.`;

const baseToolsCsv = process.env.PI_HARNESS_TOOLS || "";
const baseTools = baseToolsCsv ? baseToolsCsv.split(",").map((s) => s.trim()).filter(Boolean) : [];
const tools = [...baseTools, ...kanbanToolNames];
const cwd = process.env.PI_HARNESS_CWD || undefined;

const { session } = await createAgentSession({
  model,
  sessionManager: SessionManager.inMemory(),
  authStorage,
  modelRegistry,
  tools,
  customTools: kanbanTools,
  ...(cwd ? { cwd } : {}),
});

let output = "";
// Stream the agent's live activity to fd 2 (stderr) so the KanbanAI live tail
// (livetail.Store, mirrored from the harness pipe) shows what the agent is
// actually doing — assistant prose, reasoning, and tool calls — in real time,
// not just this wrapper's own log lines. writeSync flushes synchronously.
let lastBlockType = null;
function emit(s) { try { writeSync(2, s); } catch {} }
session.subscribe((event) => {
  if (event.type !== "message_update") return;
  const ev = event.assistantMessageEvent;
  if (!ev) return;
  switch (ev.type) {
    case "text_delta":
      emit(ev.delta);
      output += ev.delta;
      lastBlockType = "text";
      break;
    case "thinking_delta":
      // Reasoning models (e.g. deepseek) emit most of their work as thinking —
      // stream it too so the operator sees the agent's reasoning live.
      emit(ev.delta);
      lastBlockType = "thinking";
      break;
    case "text_start":
    case "thinking_start":
      // separate consecutive blocks for readability
      if (lastBlockType && lastBlockType !== ev.type.replace("_start", "")) emit("\n");
      break;
    case "toolcall_end": {
      const name = ev.toolCall?.name;
      if (name) emit(`\n\n▸ tool: ${name}\n`);
      lastBlockType = "tool";
      break;
    }
  }
});

try {
  await session.prompt(framedPrompt);
} catch (err) {
  console.error(`pi-harness: agent prompt failed: ${err?.message || err}`);
  // Even on failure, try to persist whatever was produced so the lane has context.
  try { await persistOutput(); } catch {}
  try { await session.dispose(); } catch {}
  process.exit(1);
} finally {
  try { await session.dispose(); } catch {}
}

console.error(`pi-harness: agent finished (output ${output.length} chars, finalized=${phaseFinalized}, outputSaved=${outputSaved})`);

// --- bridge: persistence guarantee + finalization fallback -------------------
async function persistOutput() {
  if (outputSaved || !output.trim()) return;
  try {
    await api("PUT", `/tasks/${taskId}/output`, {
      phase: phase,
      output: output.trim(),
      summary: "",
    });
    console.error(`pi-harness: auto-saved full phase output (${output.trim().length} chars)`);
  } catch (e) {
    console.error(`pi-harness: auto-save output failed: ${e.message}`);
  }
}

async function fallbackComplete() {
  // The agent may have finalized the phase itself by calling complete_phase /
  // reopen_phase directly (the tools set phaseFinalized), in which case we skip.
  // But if it bypassed the tools, phaseFinalized stays null even though the lane
  // already moved. Re-fetch the task: if the current phase is no longer the one
  // we were running, the agent already advanced/reopened it — do nothing.
  try {
    const r = await api("GET", `/tasks/${taskId}`);
    const cur = (r.data && (r.data.task?.current_phase ?? r.data.current_phase)) || (r.data && r.data.current_phase);
    if (cur && cur !== phase) {
      console.error(`pi-harness: skipping auto-complete — phase already ${cur} (agent finalized via tool)`);
      return;
    }
  } catch (e) {
    console.error(`pi-harness: pre-complete task check failed: ${e.message}`);
  }
  const summary = output.trim().slice(-800);
  try {
    await api("POST", `/tasks/${taskId}/complete`, { phase, summary });
    console.error("pi-harness: auto-completed phase (agent did not finalize)");
  } catch (e) {
    console.error(`pi-harness: auto-complete failed: ${e.message}`);
    // Don't fail the process: a spurious 400 (phase already advanced via a path
    // we couldn't detect) must not trigger a retry storm that exhausts attempts.
  }
}

try {
  // Always ensure the full work product is persisted (unless the agent already did).
  await persistOutput();
  // If the agent did not move the lane itself, advance it so it never hangs.
  if (!phaseFinalized) {
    await fallbackComplete();
  }
} catch (e) {
  console.error(`pi-harness: ${e.message}`);
  process.exit(1);
}

console.error(`pi-harness: done (finalized=${phaseFinalized})`);
process.exit(0);