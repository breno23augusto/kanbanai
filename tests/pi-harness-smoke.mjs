// pi-harness-smoke.mjs — Smoke test for scripts/pi-harness.mjs
//
// The pi harness imports the pi SDK from `${PI_PKG_DIR}/dist/index.js` and
// typebox from `${PI_PKG_DIR}/node_modules/typebox/build/index.mjs`. Rather
// than depend on a real model + provider (slow, non-deterministic, networked),
// this test installs a *mock SDK* at a temp PI_PKG_DIR whose `createAgentSession`
// replays a scripted event sequence (text/thinking deltas + tool calls) and
// invokes the harness's own customTool `execute` functions — which call the REST
// API. A tiny in-process mock HTTP server stands in for KanbanAI's REST API,
// recording every request so we can assert the harness made the right calls.
//
// Coverage (mirrors claude-harness-smoke.mjs):
//   1. Basic execution — starts, logs task/phase/model, exits 0
//   2. CustomTools registered — all 6 KanbanAI tools passed to createAgentSession
//   3. Tools array — base tools (PI_HARNESS_TOOLS) + custom tool names
//   4. Live tail streaming — stderr contains text_delta + ▸ tool markers
//   5. Happy path — agent calls complete_phase → POST /complete; no auto-complete
//   6. Safety-net auto-save — no update_task_output → harness PUTs full output
//   7. Safety-net auto-complete — no finalize → harness POSTs /complete
//   8. Safety-net tolerant — phase already advanced → skips auto-complete
//   9. Reopen path — agent calls reopen_phase → POST /reopen; no auto-complete
//  10. Prompt failure — session.prompt throws → persists output, exits 1
//  11. Env propagation — KANBANAI_TASK_ID/PHASE/API_URL reach the harness
//  12. Cleanup — temp SDK dir deleted
//
// Usage:
//   node tests/pi-harness-smoke.mjs
//
// Exit code: 0 = all pass, 1 = any fail.

import { strict as assert } from "node:assert";
import {
  mkdtempSync, writeFileSync, rmSync, mkdirSync, existsSync, readFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { spawn } from "node:child_process";
import { createServer } from "node:http";

const __dirname = dirname(fileURLToPath(import.meta.url));
const HARNESS_PATH = join(__dirname, "..", "scripts", "pi-harness.mjs");

let passed = 0;
let failed = 0;

function test(name, fn) {
  try { fn(); passed++; console.error(`  ✓ ${name}`); }
  catch (e) { failed++; console.error(`  ✗ ${name}: ${e.message}`); }
}
const includes = (hay, needle, label) => {
  if (!hay.includes(needle)) throw new Error(`${label}: expected to include ${JSON.stringify(needle)}, got: ${hay.slice(0, 200)}`);
};

// ---------------------------------------------------------------------------
// Mock pi SDK: a temp PI_PKG_DIR with dist/index.js (ESM) + a typebox stub.
// The mock createAgentSession records its args and returns a session whose
// prompt() replays the scripted events, invoking customTool.execute for tools.
// ---------------------------------------------------------------------------
let MOCK_PKG_DIR = null;

function setupMockSdk() {
  MOCK_PKG_DIR = mkdtempSync(join(tmpdir(), "kanbanai-pi-sdk-"));
  mkdirSync(join(MOCK_PKG_DIR, "dist"), { recursive: true });
  mkdirSync(join(MOCK_PKG_DIR, "node_modules", "typebox", "build"), { recursive: true });

  // package.json makes .js files ESM so `import(.../dist/index.js)` works.
  writeFileSync(join(MOCK_PKG_DIR, "package.json"), JSON.stringify({ type: "module" }));

  // typebox stub — the harness only passes Type.* schemas to defineTool; it
  // never introspects them. Placeholder objects suffice.
  const TypeStub = `export const Type = {
    Object: (o) => ({ type: "object", ...o }),
    String: (o = {}) => ({ type: "string", ...o }),
    Array: (o, opts = {}) => ({ type: "array", items: o, ...opts }),
    Optional: (o) => ({ ...o, optional: true }),
  };`;
  writeFileSync(join(MOCK_PKG_DIR, "node_modules", "typebox", "build", "index.mjs"), TypeStub);

  // dist/index.js — the mock SDK. Runs IN THE CHILD PROCESS, so it writes its
  // captures (createAgentSession opts + the framed prompt) to files named by
  // PI_SMOKE_CAPTURE_FILE / PI_SMOKE_PROMPT_FILE, which the test reads back.
  // Scenario (PI_SMOKE_SCENARIO, JSON): { events: [...], promptThrows: bool }
  //   event types: {type:"text_delta", delta}, {type:"thinking_delta", delta},
  //                {type:"toolcall", toolName, params}
  const sdkSrc = `
import { writeFileSync } from "node:fs";
export const defineTool = (def) => def;
export const AuthStorage = { create: () => ({}) };
export const ModelRegistry = {
  create: () => ({
    find: () => null,
    getAvailable: () => [{ provider: "ollama", id: "test-model" }],
  }),
};
export const SessionManager = { inMemory: () => ({}) };
export async function createAgentSession(opts) {
  // Serialize opts (customTools have function members -> record metadata only).
  const captureFile = process.env.PI_SMOKE_CAPTURE_FILE;
  if (captureFile) {
    const serializable = {
      model: opts.model ? { provider: opts.model.provider, id: opts.model.id } : null,
      tools: opts.tools || [],
      cwd: opts.cwd || null,
      customTools: (opts.customTools || []).map((t) => ({
        name: t.name, label: t.label, hasExecute: typeof t.execute === "function",
      })),
    };
    try { writeFileSync(captureFile, JSON.stringify(serializable)); } catch {}
  }
  const customTools = opts.customTools || [];
  const scenario = JSON.parse(process.env.PI_SMOKE_SCENARIO || '{"events":[]}');
  let subscriber = () => {};
  const session = {
    subscribe: (fn) => { subscriber = fn; },
    async prompt(text) {
      const pf = process.env.PI_SMOKE_PROMPT_FILE;
      if (pf) { try { writeFileSync(pf, text); } catch {} }
      const emit = (assistantMessageEvent) =>
        subscriber({ type: "message_update", assistantMessageEvent });
      for (const ev of scenario.events || []) {
        if (ev.type === "text_start" || ev.type === "thinking_start") {
          emit({ type: ev.type });
        } else if (ev.type === "text_delta") {
          emit({ type: "text_delta", delta: ev.delta });
        } else if (ev.type === "thinking_delta") {
          emit({ type: "thinking_delta", delta: ev.delta });
        } else if (ev.type === "toolcall") {
          // Emit the toolcall_end marker for the live tail, then actually run
          // the harness's customTool.execute so the REST bridge fires and the
          // phaseFinalized / outputSaved flags flip inside the harness.
          emit({ type: "toolcall_end", toolCall: { name: ev.toolName } });
          const tool = customTools.find((t) => t.name === ev.toolName);
          if (tool) {
            try { await tool.execute("smoke-call-id", ev.params || {}); }
            catch (e) { /* swallow; the harness's execute() returns errors, not throws */ }
          }
        }
      }
      if (scenario.promptThrows) throw new Error("simulated prompt failure");
    },
    async dispose() {},
  };
  return { session };
};
`;
  writeFileSync(join(MOCK_PKG_DIR, "dist", "index.js"), sdkSrc);
}

function cleanupMockSdk() {
  if (MOCK_PKG_DIR) try { rmSync(MOCK_PKG_DIR, { recursive: true, force: true }); } catch {}
}

// ---------------------------------------------------------------------------
// Mock KanbanAI REST server: records requests, returns canned responses, and
// holds mutable state (currentPhase) so the tolerant-guard test can simulate
// "phase already advanced".
// ---------------------------------------------------------------------------
function startMockApi(initialPhase = "doing") {
  const state = { currentPhase: initialPhase, requests: [] };
  const server = createServer((req, res) => {
    let body = "";
    req.on("data", (c) => { body += c; });
    req.on("end", () => {
      const url = new URL(req.url, "http://localhost");
      const path = url.pathname;
      let parsed = null;
      try { parsed = body ? JSON.parse(body) : null; } catch {}
      state.requests.push({ method: req.method, path, body: parsed });

      const send = (code, json) => {
        res.writeHead(code, { "Content-Type": "application/json" });
        res.end(JSON.stringify(json));
      };

      // GET /api/v1/tasks/:id → task with current_phase from state
      const getTask = /^\/api\/v1\/tasks\/[^/]+$/;
      const complete = /^\/api\/v1\/tasks\/[^/]+\/complete$/;
      const reopen = /^\/api\/v1\/tasks\/[^/]+\/reopen$/;
      const output = /^\/api\/v1\/tasks\/[^/]+\/output$/;
      const subtasks = /^\/api\/v1\/tasks\/[^/]+\/subtasks$/;

      if (req.method === "GET" && getTask.test(path))
        return send(200, { data: { task: { id: "smoke-task", current_phase: state.currentPhase } } });
      if (req.method === "PUT" && output.test(path))
        return send(200, { data: { ok: true } });
      if (req.method === "POST" && complete.test(path))
        return send(200, { data: { ok: true, advanced: true } });
      if (req.method === "POST" && reopen.test(path))
        return send(200, { data: { ok: true, reopened: true } });
      if (req.method === "POST" && subtasks.test(path))
        return send(200, { data: { subtasks: (parsed?.subtasks || []).map((_, i) => ({ id: `sub-${i}`, title: _.title })) } });
      if (req.method === "PATCH" && /^\/api\/v1\/tasks\/[^/]+\/subtasks\/[^/]+$/.test(path))
        return send(200, { data: { ok: true } });
      // default
      send(200, { data: { ok: true } });
    });
  });
  return new Promise((resolve) => {
    server.listen(0, "127.0.0.1", () => {
      const port = server.address().port;
      resolve({
        port,
        url: `http://127.0.0.1:${port}`,
        state,
        close: () => new Promise((r) => server.close(() => r())),
      });
    });
  });
}

// ---------------------------------------------------------------------------
// Harness runner: spawns pi-harness.mjs with the mock SDK + mock API.
// ---------------------------------------------------------------------------
function runHarness({ scenario, prompt = "test prompt", phase = "doing", apiState, envOverrides = {} }) {
  return new Promise((resolve) => {
    const captureDir = mkdtempSync(join(tmpdir(), "kanbanai-pi-cap-"));
    const captureFile = join(captureDir, "capture.json");
    const promptFile = join(captureDir, "prompt.txt");
    const env = {
      ...process.env,
      PI_PKG_DIR: MOCK_PKG_DIR,
      KANBANAI_TASK_ID: "smoke-task-001",
      KANBANAI_PHASE: phase,
      KANBANAI_API_URL: apiState ? `http://127.0.0.1:${apiState.port}` : "http://127.0.0.1:1",
      KANBANAI_MCP_URL: "http://127.0.0.1:18401/mcp/sse",
      KANBANAI_API_BASE_URL: apiState ? `http://127.0.0.1:${apiState.port}` : "http://127.0.0.1:1",
      PI_SMOKE_SCENARIO: JSON.stringify(scenario),
      PI_SMOKE_CAPTURE_FILE: captureFile,
      PI_SMOKE_PROMPT_FILE: promptFile,
      PI_HARNESS_TOOLS: "read,bash",
      PI_HARNESS_CWD: process.cwd(),
      ...envOverrides,
    };

    const child = spawn(process.execPath, [HARNESS_PATH, "--model", "ollama/test-model", "--prompt", prompt], {
      env, stdio: ["ignore", "pipe", "pipe"],
    });
    let stdout = "", stderr = "";
    child.stdout.on("data", (c) => { stdout += c; });
    child.stderr.on("data", (c) => { stderr += c; });
    child.on("close", (code) => {
      let capture = null, framedPrompt = "";
      try { capture = JSON.parse(readFileSync(captureFile, "utf-8")); } catch {}
      try { framedPrompt = readFileSync(promptFile, "utf-8"); } catch {}
      try { rmSync(captureDir, { recursive: true, force: true }); } catch {}
      resolve({ code, stdout, stderr, capture, framedPrompt });
    });
  });
}

// Scenario builders ----------------------------------------------------------
const SCN = {
  // Agent emits text + calls complete_phase itself.
  happyComplete: [
    { type: "text_delta", delta: "I have finished the work." },
    { type: "toolcall", toolName: "complete_phase", params: { task_id: "smoke-task-001", phase: "doing", summary: "done" } },
  ],
  // Agent emits text + a thinking block, no finalize → safety-net.
  noFinalize: [
    { type: "thinking_delta", delta: "Considering options..." },
    { type: "text_delta", delta: "Here is my output product." },
  ],
  // Agent calls reopen_phase.
  reopen: [
    { type: "text_delta", delta: "Found problems, sending back." },
    { type: "toolcall", toolName: "reopen_phase", params: { task_id: "smoke-task-001", target_phase: "doing", reason: "needs rework" } },
  ],
  // Agent calls update_task_output then complete.
  explicitSaveAndComplete: [
    { type: "text_delta", delta: "Working..." },
    { type: "toolcall", toolName: "update_task_output", params: { task_id: "smoke-task-001", phase: "doing", output: "explicit output" } },
    { type: "toolcall", toolName: "complete_phase", params: { task_id: "smoke-task-001", phase: "doing", summary: "done" } },
  ],
  // Agent emits text, no finalize, but prompt() throws.
  promptThrows: [
    { type: "text_delta", delta: "partial output before crash" },
  ],
};

// ---------------------------------------------------------------------------
async function main() {
  console.error("pi-harness-smoke: setting up mock SDK...");
  setupMockSdk();

  // =========================================================================
  // 1. Basic execution
  // =========================================================================
  console.error("\n--- Basic Execution ---");
  {
    const api = await startMockApi("doing");
    const r = await runHarness({ scenario: { events: SCN.happyComplete }, apiState: api });
    test("harness exits 0 on happy path", () => assert.equal(r.code, 0, `stderr: ${r.stderr}`));
    test("logs task ID on startup", () => includes(r.stderr, "smoke-task-001", "task id"));
    test("logs phase on startup", () => includes(r.stderr, "phase=doing", "phase"));
    test("logs resolved model on startup", () => includes(r.stderr, "model=ollama/test-model", "model"));
    await api.close();
  }

  // =========================================================================
  // 2. CustomTools registered (captured by the mock SDK)
  // =========================================================================
  console.error("\n--- CustomTools Registered ---");
  {
    const api = await startMockApi("doing");
    const r = await runHarness({ scenario: { events: SCN.happyComplete }, apiState: api });
    const opts = r.capture;
    test("createAgentSession received customTools array", () => assert.ok(Array.isArray(opts?.customTools)));
    const names = (opts?.customTools || []).map((t) => t.name).sort();
    test("all 6 KanbanAI tools registered", () => assert.deepEqual(
      names,
      ["complete_phase", "create_subtasks", "get_task", "reopen_phase", "update_subtask_status", "update_task_output"].sort(),
    ));
    test("each tool has an execute function", () => {
      for (const t of opts.customTools) assert.equal(t.hasExecute, true, `${t.name}.execute`);
    });
    await api.close();
  }

  // =========================================================================
  // 3. Tools array = base tools (PI_HARNESS_TOOLS) + custom tool names
  // =========================================================================
  console.error("\n--- Tools Array ---");
  {
    const api = await startMockApi("doing");
    const r = await runHarness({ scenario: { events: SCN.happyComplete }, apiState: api });
    const opts = r.capture;
    test("tools array includes base tools from PI_HARNESS_TOOLS", () => {
      includes(opts.tools.join(","), "read", "base tools");
      includes(opts.tools.join(","), "bash", "base tools");
    });
    test("tools array includes all custom tool names", () => {
      const t = opts.tools.join(",");
      for (const n of ["get_task", "create_subtasks", "update_subtask_status", "update_task_output", "complete_phase", "reopen_phase"])
        includes(t, n, "custom tool in tools array");
    });
    await api.close();
  }

  // =========================================================================
  // 4. Live tail streaming — text + thinking + tool markers reach stderr
  // =========================================================================
  console.error("\n--- Live Tail Streaming ---");
  {
    const api = await startMockApi("doing");
    const r = await runHarness({ scenario: { events: SCN.noFinalize }, apiState: api });
    test("stderr streams text_delta prose", () => includes(r.stderr, "Here is my output product.", "text"));
    test("stderr streams thinking_delta prose", () => includes(r.stderr, "Considering options...", "thinking"));
    await api.close();
  }
  {
    const api = await startMockApi("doing");
    const r = await runHarness({ scenario: { events: SCN.happyComplete }, apiState: api });
    test("stderr emits ▸ tool marker for toolcall_end", () => includes(r.stderr, "▸ tool: complete_phase", "tool marker"));
    await api.close();
  }

  // =========================================================================
  // 5. Happy path — agent calls complete_phase → POST /complete, no auto-complete
  // =========================================================================
  console.error("\n--- Happy Path: explicit complete_phase ---");
  {
    const api = await startMockApi("doing");
    const r = await runHarness({ scenario: { events: SCN.happyComplete }, apiState: api });
    test("POST /complete recorded", () => {
      assert.ok(api.state.requests.some((q) => q.method === "POST" && q.path.endsWith("/complete")), "no /complete call");
    });
    test("complete body includes phase + summary", () => {
      const q = api.state.requests.find((x) => x.method === "POST" && x.path.endsWith("/complete"));
      assert.equal(q.body.phase, "doing");
      assert.equal(q.body.summary, "done");
    });
    test("harness reports finalized=complete", () => includes(r.stderr, "finalized=complete", "finalized"));
    test("only ONE /complete call (no auto-complete double-fire)", () => {
      const n = api.state.requests.filter((q) => q.method === "POST" && q.path.endsWith("/complete")).length;
      assert.equal(n, 1, `expected 1 complete, got ${n}`);
    });
    await api.close();
  }

  // =========================================================================
  // 6. Safety-net auto-save — agent didn't call update_task_output
  // =========================================================================
  console.error("\n--- Safety-net: auto-save output ---");
  {
    const api = await startMockApi("doing");
    const r = await runHarness({ scenario: { events: SCN.noFinalize }, apiState: api });
    test("harness PUT /output (auto-saved full output)", () => {
      assert.ok(api.state.requests.some((q) => q.method === "PUT" && q.path.endsWith("/output")), "no /output PUT");
    });
    test("auto-saved body contains the agent's text", () => {
      const q = api.state.requests.find((x) => x.method === "PUT" && x.path.endsWith("/output"));
      includes(q.body.output, "Here is my output product.", "auto-saved output");
    });
    test("harness logs auto-saved message", () => includes(r.stderr, "auto-saved", "auto-save log"));
    await api.close();
  }
  {
    // Agent DID call update_task_output → harness must NOT auto-save again.
    const api = await startMockApi("doing");
    const r = await runHarness({ scenario: { events: SCN.explicitSaveAndComplete }, apiState: api });
    test("no second auto-save when agent already saved", () => {
      const n = api.state.requests.filter((q) => q.method === "PUT" && q.path.endsWith("/output")).length;
      assert.equal(n, 1, `expected 1 output PUT (the agent's), got ${n}`);
    });
    await api.close();
  }

  // =========================================================================
  // 7. Safety-net auto-complete — agent didn't finalize → harness POSTs /complete
  // =========================================================================
  console.error("\n--- Safety-net: auto-complete phase ---");
  {
    const api = await startMockApi("doing");
    const r = await runHarness({ scenario: { events: SCN.noFinalize }, apiState: api });
    test("harness auto-completed (POST /complete)", () => {
      assert.ok(api.state.requests.some((q) => q.method === "POST" && q.path.endsWith("/complete")), "no auto-complete");
    });
    test("harness logs auto-completed message", () => includes(r.stderr, "auto-completed", "auto-complete log"));
    test("auto-complete summary is the tail of the output", () => {
      const q = api.state.requests.find((x) => x.method === "POST" && x.path.endsWith("/complete") && x.body?.summary);
      includes(q.body.summary, "output product.", "auto-complete summary");
    });
    await api.close();
  }

  // =========================================================================
  // 8. Safety-net tolerant — phase already advanced → skip auto-complete
  // =========================================================================
  console.error("\n--- Safety-net: tolerant pre-complete guard ---");
  {
    // The mock GET /tasks/:id returns current_phase from state. Set it to
    // something OTHER than the running phase to simulate "already advanced".
    const api = await startMockApi("testing"); // != "doing"
    const r = await runHarness({ scenario: { events: SCN.noFinalize }, phase: "doing", apiState: api });
    test("harness logs 'skipping auto-complete'", () => includes(r.stderr, "skipping auto-complete", "skip log"));
    test("NO /complete POSTed (phase already advanced)", () => {
      const n = api.state.requests.filter((q) => q.method === "POST" && q.path.endsWith("/complete")).length;
      assert.equal(n, 0, `expected 0 complete, got ${n}`);
    });
    test("GET /tasks/:id was issued for the guard check", () => {
      assert.ok(api.state.requests.some((q) => q.method === "GET"), "no GET guard");
    });
    await api.close();
  }

  // =========================================================================
  // 9. Reopen path — agent calls reopen_phase
  // =========================================================================
  console.error("\n--- Reopen Path ---");
  {
    const api = await startMockApi("validating");
    const r = await runHarness({ scenario: { events: SCN.reopen }, phase: "validating", apiState: api });
    test("POST /reopen recorded", () => {
      assert.ok(api.state.requests.some((q) => q.method === "POST" && q.path.endsWith("/reopen")), "no /reopen");
    });
    test("reopen body has target_phase + reason", () => {
      const q = api.state.requests.find((x) => x.method === "POST" && x.path.endsWith("/reopen"));
      assert.equal(q.body.target_phase, "doing");
      includes(q.body.reason, "rework", "reopen reason");
    });
    test("harness reports finalized=reopen", () => includes(r.stderr, "finalized=reopen", "finalized"));
    test("NO auto-complete after reopen", () => {
      const n = api.state.requests.filter((q) => q.method === "POST" && q.path.endsWith("/complete")).length;
      assert.equal(n, 0, `expected 0 complete after reopen, got ${n}`);
    });
    await api.close();
  }

  // =========================================================================
  // 10. Prompt failure — session.prompt throws → persist output, exit 1
  // =========================================================================
  console.error("\n--- Prompt Failure ---");
  {
    const api = await startMockApi("doing");
    const r = await runHarness({ scenario: { events: SCN.promptThrows, promptThrows: true }, apiState: api });
    test("harness exits 1 on prompt failure", () => assert.equal(r.code, 1, `code=${r.code}`));
    test("logs prompt failure", () => includes(r.stderr, "agent prompt failed", "failure log"));
    test("still persisted partial output", () => {
      assert.ok(api.state.requests.some((q) => q.method === "PUT" && q.path.endsWith("/output")), "no output PUT on failure");
      const q = api.state.requests.find((x) => x.method === "PUT" && x.path.endsWith("/output"));
      includes(q.body.output, "partial output before crash", "partial output");
    });
    await api.close();
  }

  // =========================================================================
  // 11. Env propagation — KANBANAI_* reach the harness process
  // =========================================================================
  console.error("\n--- Environment Propagation ---");
  {
    const api = await startMockApi("doing");
    const r = await runHarness({
      scenario: { events: SCN.happyComplete }, phase: "planning",
      apiState: api, envOverrides: { KANBANAI_TASK_ID: "env-task-42", KANBANAI_PHASE: "planning" },
    });
    test("framed prompt was passed to session.prompt", () => {
      assert.ok(typeof r.framedPrompt === "string" && r.framedPrompt.length > 0, "no framed prompt captured");
    });
    test("framed prompt includes the KanbanAI lifecycle reminder", () => {
      includes(r.framedPrompt, "KanbanAI phase worker", "framing");
    });
    test("framed prompt contains the original prompt text", () => {
      includes(r.framedPrompt, "test prompt", "original prompt");
    });
    await api.close();
  }

  // =========================================================================
  // 12. Cleanup — temp SDK dir deleted
  // =========================================================================
  console.error("\n--- Cleanup ---");
  cleanupMockSdk();
  test("temp mock SDK dir deleted", () => assert.ok(!existsSync(MOCK_PKG_DIR), "SDK dir still exists"));

  // =========================================================================
  console.error("\n===============================================================");
  console.error(`Results: ${passed} passed, ${failed} failed, ${passed + failed} total`);
  console.error("===============================================================");
  process.exit(failed ? 1 : 0);
}

main().catch((e) => { console.error("fatal:", e); process.exit(1); });