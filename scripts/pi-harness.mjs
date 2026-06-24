// pi-harness.mjs — KanbanAI harness implemented with the pi SDK.
//
// Bridge for non-MCP harnesses: pi has no built-in MCP support, so instead of
// the harness calling the complete_phase MCP tool, this script runs the agent
// for the current phase and then POSTs to KanbanAI's REST complete endpoint,
// which invokes the same AdvancePhase use case and lets the orchestrator
// advance the lane via the phase.<phase>.completed event.
//
// Invoked by scripts/pi-harness.sh as: node pi-harness.mjs --model <m> --prompt <p>
// Env: KANBANAI_TASK_ID, KANBANAI_API_URL, PI_PKG_DIR, PI_HARNESS_MODEL,
//      PI_HARNESS_TOOLS (csv, optional), PI_HARNESS_CWD (optional).

const pkgDir = process.env.PI_PKG_DIR;
if (!pkgDir) {
  console.error("pi-harness: PI_PKG_DIR not set");
  process.exit(2);
}

const sdk = await import(`${pkgDir}/dist/index.js`);
const { AuthStorage, ModelRegistry, SessionManager, createAgentSession } = sdk;

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
const apiUrl = (process.env.KANBANAI_API_URL || "http://localhost:8080").replace(/\/$/, "");
if (!taskId) {
  console.error("pi-harness: KANBANAI_TASK_ID not set");
  process.exit(2);
}

// --- resolve model ----------------------------------------------------------
const authStorage = AuthStorage.create();
const modelRegistry = ModelRegistry.create(authStorage);

function resolveModel(pattern) {
  const slash = pattern.indexOf("/");
  let provider = slash >= 0 ? pattern.slice(0, slash) : pattern;
  let id = slash >= 0 ? pattern.slice(slash + 1) : "";
  let model = id ? modelRegistry.find(provider, id) : null;
  if (model) return model;
  // fuzzy fallback over available models
  const lower = pattern.toLowerCase();
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

console.error(`pi-harness: task=${taskId} model=${model.provider}/${model.id}`);

// --- run agent --------------------------------------------------------------
// Completion of the phase is handled by this wrapper once the agent finishes,
// so we append a framing note and ask the agent to summarize its result.
const framedPrompt =
  `${prompt}\n\n` +
  `---\nYou are running as an automated KanbanAI phase worker. ` +
  `Do not attempt to call any MCP tools; phase completion is signaled ` +
  `automatically by the harness when you finish. Perform the requested work ` +
  `for this phase and end with a concise summary of what you produced or concluded.`;

const toolsCsv = process.env.PI_HARNESS_TOOLS || "";
const tools = toolsCsv ? toolsCsv.split(",").map((s) => s.trim()).filter(Boolean) : [];
const cwd = process.env.PI_HARNESS_CWD || undefined;

const { session } = await createAgentSession({
  model,
  sessionManager: SessionManager.inMemory(),
  authStorage,
  modelRegistry,
  tools,
  ...(cwd ? { cwd } : {}),
});

let output = "";
session.subscribe((event) => {
  if (event.type === "message_update" && event.assistantMessageEvent?.type === "text_delta") {
    output += event.assistantMessageEvent.delta;
  }
});

try {
  await session.prompt(framedPrompt);
} catch (err) {
  console.error(`pi-harness: agent prompt failed: ${err?.message || err}`);
  try { await session.dispose(); } catch {}
  process.exit(1);
} finally {
  try { await session.dispose(); } catch {}
}

console.error(`pi-harness: agent finished, completing phase (output ${output.length} chars)`);

// --- bridge to KanbanAI: complete the current phase -------------------------
const summary = output.trim().slice(-800);

async function completePhase() {
  const res = await fetch(`${apiUrl}/api/v1/tasks/${taskId}/complete`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ summary }),
  });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`complete failed: ${res.status} ${res.statusText} ${body}`);
  }
}

try {
  await completePhase();
} catch (err) {
  console.error(`pi-harness: ${err.message}`);
  process.exit(1);
}

console.error("pi-harness: phase completed");
process.exit(0);