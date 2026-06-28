// claude-harness.mjs — KanbanAI harness for Claude Code (headless).
//
// Unlike the pi harness (which has no MCP and shims KanbanAI tools over REST as
// pi customTools), Claude Code speaks MCP natively. So this harness just points
// Claude at the KanbanAI MCP server (SSE at KANBANAI_MCP_URL) — the agent calls
// the real create_subtasks / update_subtask_status / update_task_output /
// complete_phase / reopen_phase / get_task / report_progress tools directly.
//
// What this harness adds on top of `claude -p`:
//   1. Live tail — parses --output-format=stream-json and pretty-prints the
//      agent's thinking, text, and tool calls to stderr in real time so the
//      KanbanAI livetail.Store (which mirrors the process pipe) shows the work.
//   2. Persistence + finalization safety-net — if the agent exits without
//      calling update_task_output / complete_phase, this harness persists the
//      captured assistant text and advances the lane so it never hangs.
//
// Invoked by scripts/claude-harness.sh as:
//   node claude-harness.mjs --model <m> --prompt <p>
// Env: KANBANAI_TASK_ID, KANBANAI_PHASE, KANBANAI_MCP_URL, KANBANAI_API_BASE_URL,
//      PI_HARNESS_CWD (optional).

import { writeSync } from "node:fs";
import { spawn } from "node:child_process";
import { mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

// --- args + env --------------------------------------------------------------
let model = "";
let prompt = "";
for (let i = 2; i < process.argv.length; i++) {
  const a = process.argv[i];
  if (a === "--model") model = process.argv[++i] ?? "";
  else if (a === "--prompt") prompt = process.argv[++i] ?? "";
}
if (!prompt) { console.error("claude-harness: --prompt is required"); process.exit(2); }

const taskId = process.env.KANBANAI_TASK_ID;
const phase = process.env.KANBANAI_PHASE || "doing";
const mcpUrl = process.env.KANBANAI_MCP_URL || "http://localhost:18401/mcp/sse";
const apiUrl = (process.env.KANBANAI_API_BASE_URL || "http://localhost:18400/api/v1").replace(/\/api\/v1\/?$/, "");
const cwd = process.env.PI_HARNESS_CWD || process.cwd();

if (!taskId) { console.error("claude-harness: KANBANAI_TASK_ID not set"); process.exit(2); }

console.error(`claude-harness: task=${taskId} phase=${phase} model=${model || "(claude default)"} mcp=${mcpUrl}`);

// --- MCP config (temp file) --------------------------------------------------
// Point Claude Code at the KanbanAI MCP server. Tools are namespaced
// mcp__kanbanai__<tool>; the prompt refers to them by bare name, which Claude
// resolves. MCP security validates every tool's task_id against KANBANAI_TASK_ID.
const tmpDir = mkdtempSync(join(tmpdir(), "kanbanai-claude-"));
const mcpConfigPath = join(tmpDir, "mcp.json");
writeFileSync(mcpConfigPath, JSON.stringify({
  mcpServers: {
    kanbanai: { type: "sse", url: mcpUrl },
  },
}));
const cleanup = () => { try { rmSync(tmpDir, { recursive: true, force: true }); } catch {} };
process.on("exit", cleanup);

// --- live tail emitter -------------------------------------------------------
function emit(s) { try { writeSync(2, s); } catch {} }

let output = "";            // accumulated assistant text (for safety-net persistence)
let phaseFinalized = null;  // null | "complete" | "reopen"
let lastBlockType = null;

// --- stream-json parser ------------------------------------------------------
// Claude Code with --output-format=stream-json --verbose emits one JSON object
// per line: system/*, assistant (content blocks), user (tool_result), result.
function handleEvent(obj) {
  const t = obj.type;
  if (t === "system") return; // init/thinking_tokens — noisy, skip
  if (t === "assistant" || t === "user") {
    const content = obj?.message?.content;
    if (!Array.isArray(content)) return;
    for (const block of content) {
      if (!block || typeof block !== "object") continue;
      switch (block.type) {
        case "thinking": {
          const txt = block.thinking || "";
          if (txt) { if (lastBlockType !== "thinking") emit("\n"); emit(txt); lastBlockType = "thinking"; }
          break;
        }
        case "text": {
          const txt = block.text || "";
          if (txt) { if (lastBlockType && lastBlockType !== "text") emit("\n"); emit(txt); output += txt; lastBlockType = "text"; }
          break;
        }
        case "tool_use": {
          const name = block.name || "tool";
          // MCP tools arrive as mcp__kanbanai__complete_phase etc.; normalize.
          const bare = String(name).replace(/^mcp__kanbanai__/, "");
          emit(`\n\n▸ tool: ${bare}\n`);
          lastBlockType = "tool";
          if (bare === "complete_phase") phaseFinalized = "complete";
          else if (bare === "reopen_phase") phaseFinalized = "reopen";
          break;
        }
        case "tool_result": {
          // Compact one-liner so the operator sees tool outcomes without noise.
          let s = "";
          if (Array.isArray(block.content)) s = block.content.map((c) => c?.text ?? "").join("");
          else if (typeof block.content === "string") s = block.content;
          s = (s || "").replace(/\s+/g, " ").trim().slice(0, 200);
          if (s) emit(`  ↳ ${s}\n`);
          lastBlockType = "result";
          break;
        }
      }
    }
    return;
  }
  if (t === "result") {
    // Final result: capture any trailing text not seen as a content block.
    const r = obj.result || "";
    if (r && !output.includes(r)) output += (output ? "\n" : "") + r;
    return;
  }
}

// --- spawn claude ------------------------------------------------------------
const args = [
  "-p", prompt,
  "--output-format", "stream-json",
  "--verbose",
  "--mcp-config", mcpConfigPath,
  "--dangerously-skip-permissions",
  "--permission-mode", "bypassPermissions",
  "--add-dir", cwd,
];
if (model) args.push("--model", model);

const child = spawn("claude", args, { cwd, stdio: ["ignore", "pipe", "inherit"] });

let stdoutBuf = "";
child.stdout.on("data", (chunk) => {
  stdoutBuf += chunk.toString();
  // stream-json is newline-delimited JSON; process complete lines eagerly so the
  // live tail updates in real time (don't wait for process exit).
  let nl;
  while ((nl = stdoutBuf.indexOf("\n")) >= 0) {
    const line = stdoutBuf.slice(0, nl).trim();
    stdoutBuf = stdoutBuf.slice(nl + 1);
    if (!line) continue;
    try { handleEvent(JSON.parse(line)); }
    catch { /* non-JSON line (e.g. a stray log) — ignore */ }
  }
});

const exitCode = await new Promise((resolve) => {
  child.on("error", (err) => {
    console.error(`claude-harness: failed to spawn claude: ${err.message}`);
    console.error("  ensure 'claude' is installed and on PATH");
    resolve(1);
  });
  child.on("close", resolve);
});

console.error(`claude-harness: agent finished (output ${output.length} chars, finalized=${phaseFinalized})`);

// --- safety-net: persist + finalize (mirrors pi-harness) ----------------------
async function api(method, path, body) {
  const res = await fetch(`${apiUrl}/api/v1${path}`, {
    method,
    headers: { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await res.text().catch(() => "");
  if (!res.ok) throw new Error(`${method} ${path} failed: ${res.status} ${res.statusText} ${text}`);
  return text ? JSON.parse(text) : {};
}

async function persistOutput() {
  if (!output.trim()) return;
  try {
    await api("PUT", `/tasks/${taskId}/output`, { phase, output: output.trim(), summary: "" });
    console.error(`claude-harness: auto-saved phase output (${output.trim().length} chars)`);
  } catch (e) {
    console.error(`claude-harness: auto-save output failed: ${e.message}`);
  }
}

async function fallbackComplete() {
  // If the agent already advanced/reopened the phase via MCP tools, the task's
  // current_phase will differ from `phase` — skip (tolerant of all finalization paths).
  try {
    const r = await api("GET", `/tasks/${taskId}`);
    const cur = r.data && (r.data.task?.current_phase ?? r.data.current_phase);
    if (cur && cur !== phase) {
      console.error(`claude-harness: skipping auto-complete — phase already ${cur} (agent finalized via MCP)`);
      return;
    }
  } catch (e) {
    console.error(`claude-harness: pre-complete task check failed: ${e.message}`);
  }
  const summary = output.trim().slice(-800);
  try {
    await api("POST", `/tasks/${taskId}/complete`, { phase, summary });
    console.error("claude-harness: auto-completed phase (agent did not finalize)");
  } catch (e) {
    // Don't fail the process: a spurious 400 (phase already advanced) must not
    // trigger a retry storm that exhausts attempts.
    console.error(`claude-harness: auto-complete failed: ${e.message}`);
  }
}

try {
  await persistOutput();
  if (!phaseFinalized) await fallbackComplete();
} catch (e) {
  console.error(`claude-harness: ${e.message}`);
}

console.error(`claude-harness: done (finalized=${phaseFinalized})`);
process.exit(exitCode === 0 ? 0 : exitCode);