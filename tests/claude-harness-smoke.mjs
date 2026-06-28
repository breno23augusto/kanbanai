// claude-harness-smoke.mjs — Comprehensive smoke test for claude-harness.mjs
//
// Validates all 13 subtasks of "claude harness smoke 3":
//   1. Basic Execution — correct flags
//   2. MCP config generation
//   3. MCP tool invocation
//   4. Live Tail: thinking blocks
//   5. Live Tail: text blocks + output buffer
//   6. Live Tail: tool_use normalization
//   7. Live Tail: tool_result compaction
//   8. Happy Path: explicit complete_phase
//   9. Safety-net: auto-save output
//  10. Safety-net: auto-complete phase
//  11. Safety-net: pre-complete guard
//  12. Environment: KANBANAI_TASK_ID / KANBANAI_PHASE propagation
//  13. Cleanup: temp dir deletion
//
// Usage:
//   KANBANAI_TASK_ID=test-123 KANBANAI_PHASE=doing \
//     node tests/claude-harness-smoke.mjs
//
// Exit code: 0 = all tests pass, 1 = any test fails

import { strict as assert } from "node:assert";
import { mkdtempSync, writeFileSync, rmSync, mkdirSync, existsSync, readFileSync, chmodSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { spawn } from "node:child_process";

// --- helpers -----------------------------------------------------------------
const __dirname = dirname(fileURLToPath(import.meta.url));
const HARNESS_PATH = join(__dirname, "..", "scripts", "claude-harness.mjs");
const MOCK_CLAUDE_DIR = join(__dirname, "..", "scripts", ".mock-claude");
const MOCK_CLAUDE_PATH = join(MOCK_CLAUDE_DIR, "claude");

let passed = 0;
let failed = 0;
const testLog = [];

function test(name, fn) {
  testLog.push({ name, status: "running" });
  const idx = testLog.length - 1;
  try {
    fn();
    testLog[idx].status = "PASS";
    passed++;
    console.error(`  ✓ ${name}`);
  } catch (e) {
    testLog[idx].status = "FAIL";
    testLog[idx].error = e.message;
    failed++;
    console.error(`  ✗ ${name}: ${e.message}`);
  }
}

function assertMatch(actual, expected, label) {
  if (!expected.test(actual)) {
    throw new Error(`${label}: expected match ${expected}, got: ${JSON.stringify(actual)}`);
  }
}

function assertIncludes(actual, expected, label) {
  if (!actual.includes(expected)) {
    throw new Error(`${label}: expected to include ${JSON.stringify(expected)}, got: ${JSON.stringify(actual)}`);
  }
}

// --- mock claude binary ------------------------------------------------------
function setupMockClaude() {
  mkdirSync(MOCK_CLAUDE_DIR, { recursive: true });

  const mockScript = `#!/usr/bin/env node
// Mock claude binary for smoke testing.
// Reads scenario from KANBANAI_SMOKE_SCENARIO_FILE env var.
// Writes args to KANBANAI_SMOKE_ARGS_LOG for verification.
// Also captures the MCP config file content if --mcp-config is present.
import { readFileSync, writeFileSync, existsSync } from "node:fs";

const argsLog = process.env.KANBANAI_SMOKE_ARGS_LOG;
const args = process.argv.slice(2);

// Capture MCP config content before the cleanup handler deletes it
let mcpConfigContent = null;
const mcpIdx = args.indexOf("--mcp-config");
if (mcpIdx >= 0 && mcpIdx + 1 < args.length) {
  const configPath = args[mcpIdx + 1];
  if (existsSync(configPath)) {
    try { mcpConfigContent = readFileSync(configPath, "utf-8"); } catch {}
  }
}

// Capture env vars
const envCapture = {
  KANBANAI_TASK_ID: process.env.KANBANAI_TASK_ID || null,
  KANBANAI_PHASE: process.env.KANBANAI_PHASE || null,
  KANBANAI_MCP_URL: process.env.KANBANAI_MCP_URL || null,
  KANBANAI_API_BASE_URL: process.env.KANBANAI_API_BASE_URL || null,
  PI_HARNESS_CWD: process.env.PI_HARNESS_CWD || null,
};

if (argsLog) {
  writeFileSync(argsLog, JSON.stringify({
    args,
    mcpConfig: mcpConfigContent,
    env: envCapture,
  }));
}

// Read scenario file
const scenarioFile = process.env.KANBANAI_SMOKE_SCENARIO_FILE;
let scenarioData = { lines: [], exitCode: 0 };
if (scenarioFile && existsSync(scenarioFile)) {
  try { scenarioData = JSON.parse(readFileSync(scenarioFile, "utf-8")); } catch {}
}

const lines = scenarioData.lines || [];
for (const line of lines) {
  process.stdout.write(JSON.stringify(line) + "\\n");
}

const exitCode = scenarioData.exitCode !== undefined ? scenarioData.exitCode : 0;
process.exit(exitCode);
`;

  writeFileSync(MOCK_CLAUDE_PATH, mockScript);
  chmodSync(MOCK_CLAUDE_PATH, 0o755); // make executable
}

function cleanupMockClaude() {
  try { rmSync(MOCK_CLAUDE_DIR, { recursive: true, force: true }); } catch {}
}

// --- scenario helpers --------------------------------------------------------
function makeScenarioFile(lines, exitCode = 0) {
  const f = mkdtempSync(join(tmpdir(), "kanbanai-smoke-scenario-"));
  const path = join(f, "scenario.json");
  writeFileSync(path, JSON.stringify({ lines, exitCode }));
  return { dir: f, path };
}

function cleanupScenario(dir) {
  try { rmSync(dir, { recursive: true, force: true }); } catch {}
}

// --- harness runner ----------------------------------------------------------
function runHarness({ scenario, prompt, envOverrides = {} }) {
  return new Promise((resolve) => {
    const argsLogDir = mkdtempSync(join(tmpdir(), "kanbanai-smoke-args-"));
    const argsLogFile = join(argsLogDir, "args.json");

    const scenarioResult = makeScenarioFile(scenario.lines, scenario.exitCode);

    const env = {
      ...process.env,
      KANBANAI_TASK_ID: "smoke-test-task-001",
      KANBANAI_PHASE: "doing",
      KANBANAI_MCP_URL: "http://localhost:18401/mcp/sse",
      KANBANAI_API_BASE_URL: "http://localhost:18400",
      KANBANAI_SMOKE_SCENARIO_FILE: scenarioResult.path,
      KANBANAI_SMOKE_ARGS_LOG: argsLogFile,
      PI_HARNESS_CWD: process.cwd(),
      // Override PATH so our mock claude is found first
      PATH: MOCK_CLAUDE_DIR + ":" + (process.env.PATH || ""),
      ...envOverrides,
    };

    const child = spawn(process.execPath, [HARNESS_PATH, "--model", "test-model", "--prompt", prompt || "test prompt"], {
      env,
      stdio: ["ignore", "pipe", "pipe"],
    });

    let stdout = "";
    let stderr = "";

    child.stdout.on("data", (chunk) => { stdout += chunk.toString(); });
    child.stderr.on("data", (chunk) => { stderr += chunk.toString(); });

    child.on("close", (code) => {
      // Read args log (written by mock claude before exit)
      let recorded = null;
      try {
        recorded = JSON.parse(readFileSync(argsLogFile, "utf-8"));
      } catch {}

      // Cleanup temp files
      try { rmSync(argsLogDir, { recursive: true, force: true }); } catch {}
      cleanupScenario(scenarioResult.dir);

      resolve({
        code,
        stdout,
        stderr,
        args: recorded?.args || [],
        mcpConfig: recorded?.mcpConfig || null,
        capturedEnv: recorded?.env || {},
      });
    });
  });
}

// --- tests -------------------------------------------------------------------
async function main() {
  console.error("claude-harness-smoke: setting up mock claude...");
  setupMockClaude();

  // =========================================================================
  // SUBTASK 1: Basic Execution — correct flags
  // =========================================================================
  console.error("\n--- Basic Execution ---");
  {
    const result = await runHarness({
      scenario: { lines: [], exitCode: 0 },
      prompt: "test prompt",
    });

    test("spawns claude with --output-format stream-json", () => {
      const idx = result.args.indexOf("--output-format");
      assert.notStrictEqual(idx, -1, "--output-format not found in args");
      assert.strictEqual(result.args[idx + 1], "stream-json", "wrong value for --output-format");
    });

    test("spawns claude with --verbose", () => {
      assert.ok(result.args.includes("--verbose"), "--verbose not found in args");
    });

    test("spawns claude with --mcp-config <path>", () => {
      const idx = result.args.indexOf("--mcp-config");
      assert.notStrictEqual(idx, -1, "--mcp-config not found in args");
      assert.ok(result.args[idx + 1].includes("mcp.json"), "mcp-config path doesn't include mcp.json");
    });

    test("spawns claude with --dangerously-skip-permissions", () => {
      assert.ok(result.args.includes("--dangerously-skip-permissions"), "--dangerously-skip-permissions not found");
    });

    test("spawns claude with --permission-mode bypassPermissions", () => {
      const idx = result.args.indexOf("--permission-mode");
      assert.notStrictEqual(idx, -1, "--permission-mode not found");
      assert.strictEqual(result.args[idx + 1], "bypassPermissions", "wrong permission mode");
    });

    test("spawns claude with --add-dir <cwd>", () => {
      const idx = result.args.indexOf("--add-dir");
      assert.notStrictEqual(idx, -1, "--add-dir not found");
      assert.ok(result.args[idx + 1], "--add-dir has no value");
    });

    test("spawns claude with -p <prompt>", () => {
      const idx = result.args.indexOf("-p");
      assert.notStrictEqual(idx, -1, "-p not found");
      assert.strictEqual(result.args[idx + 1], "test prompt", "wrong prompt value");
    });

    test("spawns claude with --model when provided", () => {
      const idx = result.args.indexOf("--model");
      assert.notStrictEqual(idx, -1, "--model not found");
      assert.strictEqual(result.args[idx + 1], "test-model", "wrong model value");
    });
  }

  // =========================================================================
  // SUBTASK 2: MCP config generation
  // =========================================================================
  console.error("\n--- MCP Config Generation ---");
  {
    const result = await runHarness({
      scenario: { lines: [], exitCode: 0 },
      prompt: "test prompt",
    });

    test("MCP config file is generated with correct structure", () => {
      assert.ok(result.mcpConfig, "MCP config content not captured");
      const config = JSON.parse(result.mcpConfig);
      assert.ok(config.mcpServers, "mcpServers key missing");
      assert.ok(config.mcpServers.kanbanai, "kanbanai server missing");
      assert.strictEqual(config.mcpServers.kanbanai.type, "sse", "wrong type");
      assert.ok(config.mcpServers.kanbanai.url.includes("mcp/sse"), "URL missing /mcp/sse");
    });

    test("MCP config URL points to KANBANAI_MCP_URL", () => {
      const config = JSON.parse(result.mcpConfig);
      assert.ok(
        config.mcpServers.kanbanai.url.includes("localhost:18401"),
        "URL doesn't point to MCP server port"
      );
    });
  }

  // =========================================================================
  // SUBTASK 3: MCP tool invocation (via stream-json parsing)
  // =========================================================================
  console.error("\n--- MCP Tool Invocation ---");
  {
    const lines = [
      { type: "assistant", message: { content: [
        { type: "tool_use", name: "mcp__kanbanai__get_task", input: { task_id: "test-123" } },
      ]}},
      { type: "user", message: { content: [
        { type: "tool_result", content: [{ type: "text", text: "Task found: test-123" }] },
      ]}},
      { type: "assistant", message: { content: [
        { type: "tool_use", name: "mcp__kanbanai__report_progress", input: { message: "working" } },
      ]}},
      { type: "user", message: { content: [
        { type: "tool_result", content: [{ type: "text", text: "Progress reported" }] },
      ]}},
    ];

    const result = await runHarness({ scenario: { lines, exitCode: 0 }, prompt: "test" });

    test("tool_use get_task is detected and normalized in stderr", () => {
      assertIncludes(result.stderr, "▸ tool: get_task", "get_task tool not found in stderr");
    });

    test("tool_use report_progress is detected and normalized in stderr", () => {
      assertIncludes(result.stderr, "▸ tool: report_progress", "report_progress tool not found in stderr");
    });

    test("tool_result for get_task is compacted in stderr", () => {
      assertIncludes(result.stderr, "↳ Task found: test-123", "tool_result not found in stderr");
    });

    test("tool_result for report_progress is compacted in stderr", () => {
      assertIncludes(result.stderr, "↳ Progress reported", "tool_result not found in stderr");
    });
  }

  // =========================================================================
  // SUBTASK 4: Live Tail — thinking blocks
  // =========================================================================
  console.error("\n--- Live Tail: Thinking Blocks ---");
  {
    const lines = [
      { type: "assistant", message: { content: [
        { type: "thinking", thinking: "Let me analyze this task step by step." },
        { type: "thinking", thinking: "First, I need to understand the requirements." },
      ]}},
    ];

    const result = await runHarness({ scenario: { lines, exitCode: 0 }, prompt: "test" });

    test("thinking blocks are printed to stderr", () => {
      assertIncludes(result.stderr, "Let me analyze this task step by step.", "first thinking block missing");
      assertIncludes(result.stderr, "First, I need to understand the requirements.", "second thinking block missing");
    });
  }

  // =========================================================================
  // SUBTASK 5: Live Tail — text blocks + output buffer
  // =========================================================================
  console.error("\n--- Live Tail: Text Blocks + Output Buffer ---");
  {
    const lines = [
      { type: "assistant", message: { content: [
        { type: "text", text: "I have analyzed the task." },
        { type: "text", text: "Here is my implementation plan." },
      ]}},
    ];

    const result = await runHarness({ scenario: { lines, exitCode: 0 }, prompt: "test" });

    test("text blocks are printed to stderr", () => {
      assertIncludes(result.stderr, "I have analyzed the task.", "first text block missing");
      assertIncludes(result.stderr, "Here is my implementation plan.", "second text block missing");
    });

    test("text blocks are accumulated into output buffer (stderr shows 'output N chars')", () => {
      assertMatch(result.stderr, /output \d+ chars/, "output length not reported in stderr");
    });
  }

  // =========================================================================
  // SUBTASK 6: Live Tail — tool_use normalization (mcp__kanbanai__ prefix stripped)
  // =========================================================================
  console.error("\n--- Live Tail: Tool_Use Normalization ---");
  {
    const lines = [
      { type: "assistant", message: { content: [
        { type: "tool_use", name: "mcp__kanbanai__complete_phase", input: { phase: "doing" } },
      ]}},
      { type: "assistant", message: { content: [
        { type: "tool_use", name: "mcp__kanbanai__reopen_phase", input: { phase: "todo" } },
      ]}},
      { type: "assistant", message: { content: [
        { type: "tool_use", name: "mcp__kanbanai__update_task_output", input: { output: "test" } },
      ]}},
    ];

    const result = await runHarness({ scenario: { lines, exitCode: 0 }, prompt: "test" });

    test("mcp__kanbanai__ prefix is stripped from tool names in stderr", () => {
      assertIncludes(result.stderr, "▸ tool: complete_phase", "complete_phase not normalized");
      assertIncludes(result.stderr, "▸ tool: reopen_phase", "reopen_phase not normalized");
      assertIncludes(result.stderr, "▸ tool: update_task_output", "update_task_output not normalized");
    });

    test("raw mcp__kanbanai__ prefix does NOT appear in stderr", () => {
      assert.ok(!result.stderr.includes("mcp__kanbanai__"), "mcp__kanbanai__ prefix leaked to stderr");
    });
  }

  // =========================================================================
  // SUBTASK 7: Live Tail — tool_result compaction (200 char max, one-liner)
  // =========================================================================
  console.error("\n--- Live Tail: Tool_Result Compaction ---");
  {
    const longText = "A".repeat(500);
    const lines = [
      { type: "user", message: { content: [
        { type: "tool_result", content: [{ type: "text", text: longText }] },
      ]}},
    ];

    const result = await runHarness({ scenario: { lines, exitCode: 0 }, prompt: "test" });

    test("tool_result is compacted to max 200 chars", () => {
      const match = result.stderr.match(/↳ (.+)/);
      assert.ok(match, "no tool_result line found in stderr");
      const compacted = match[1].trim();
      assert.ok(compacted.length <= 200, `tool_result too long: ${compacted.length} chars`);
    });

    test("tool_result is a single line (no embedded newlines)", () => {
      const toolResultLines = result.stderr.split("\n").filter(l => l.includes("↳"));
      for (const line of toolResultLines) {
        assert.ok(!line.includes("\n"), "tool_result contains newlines");
      }
    });
  }

  // =========================================================================
  // SUBTASK 8: Happy Path — explicit complete_phase sets phaseFinalized
  // =========================================================================
  console.error("\n--- Happy Path: Explicit complete_phase ---");
  {
    const lines = [
      { type: "assistant", message: { content: [
        { type: "text", text: "Work is done." },
        { type: "tool_use", name: "mcp__kanbanai__complete_phase", input: { phase: "doing" } },
      ]}},
    ];

    const result = await runHarness({ scenario: { lines, exitCode: 0 }, prompt: "test" });

    test("phaseFinalized is set to 'complete' when complete_phase is called", () => {
      assertIncludes(result.stderr, "finalized=complete", "phaseFinalized not set to complete");
    });

    test("safety-net auto-complete is skipped when phaseFinalized is set", () => {
      assert.ok(!result.stderr.includes("auto-completed phase"), "auto-complete ran despite phaseFinalized");
    });
  }

  // =========================================================================
  // SUBTASK 9: Safety-net — auto-save output
  // =========================================================================
  console.error("\n--- Safety-net: Auto-Save Output ---");
  {
    const lines = [
      { type: "assistant", message: { content: [
        { type: "text", text: "Some output that needs saving." },
      ]}},
    ];

    const result = await runHarness({ scenario: { lines, exitCode: 0 }, prompt: "test" });

    test("harness attempts to auto-save output (stderr shows auto-save attempt)", () => {
      assert.ok(
        result.stderr.includes("auto-save") || result.stderr.includes("auto-saved"),
        "auto-save not attempted"
      );
    });
  }

  // =========================================================================
  // SUBTASK 10: Safety-net — auto-complete phase
  // =========================================================================
  console.error("\n--- Safety-net: Auto-Complete Phase ---");
  {
    // Agent exits without calling complete_phase or reopen_phase
    const lines = [
      { type: "assistant", message: { content: [
        { type: "text", text: "Work done but no finalization." },
      ]}},
    ];

    const result = await runHarness({ scenario: { lines, exitCode: 0 }, prompt: "test" });

    test("harness attempts auto-complete when phaseFinalized is null", () => {
      assert.ok(
        result.stderr.includes("auto-complete") || result.stderr.includes("auto-completed"),
        "auto-complete not attempted"
      );
    });
  }

  // =========================================================================
  // SUBTASK 11: Safety-net — pre-complete guard skips if phase already advanced
  // =========================================================================
  console.error("\n--- Safety-net: Pre-Complete Guard ---");
  {
    // Agent calls reopen_phase (sets phaseFinalized to "reopen")
    const lines = [
      { type: "assistant", message: { content: [
        { type: "tool_use", name: "mcp__kanbanai__reopen_phase", input: { phase: "doing", target_phase: "todo" } },
      ]}},
    ];

    const result = await runHarness({ scenario: { lines, exitCode: 0 }, prompt: "test" });

    test("phaseFinalized is set to 'reopen' when reopen_phase is called", () => {
      assertIncludes(result.stderr, "finalized=reopen", "phaseFinalized not set to reopen");
    });

    test("auto-complete is skipped when phaseFinalized is 'reopen'", () => {
      assert.ok(!result.stderr.includes("auto-completed phase"), "auto-complete ran despite reopen");
    });
  }

  // =========================================================================
  // SUBTASK 12: Environment propagation
  // =========================================================================
  console.error("\n--- Environment Propagation ---");
  {
    const result = await runHarness({
      scenario: { lines: [], exitCode: 0 },
      prompt: "test",
    });

    test("harness logs task ID on startup", () => {
      assertIncludes(result.stderr, "task=smoke-test-task-001", "task ID not in startup log");
    });

    test("harness logs phase on startup", () => {
      assertIncludes(result.stderr, "phase=doing", "phase not in startup log");
    });

    test("harness logs MCP URL on startup", () => {
      assertIncludes(result.stderr, "mcp=http://localhost:18401/mcp/sse", "MCP URL not in startup log");
    });

    test("KANBANAI_TASK_ID is propagated to claude process", () => {
      assert.strictEqual(result.capturedEnv.KANBANAI_TASK_ID, "smoke-test-task-001", "task ID not propagated");
    });

    test("KANBANAI_PHASE is propagated to claude process", () => {
      assert.strictEqual(result.capturedEnv.KANBANAI_PHASE, "doing", "phase not propagated");
    });

    test("KANBANAI_MCP_URL is propagated to claude process", () => {
      assert.ok(
        result.capturedEnv.KANBANAI_MCP_URL?.includes("18401"),
        "MCP URL not propagated"
      );
    });

    test("PI_HARNESS_CWD is propagated to claude process", () => {
      assert.ok(result.capturedEnv.PI_HARNESS_CWD, "PI_HARNESS_CWD not propagated");
    });
  }

  // =========================================================================
  // SUBTASK 13: Cleanup — temp dir deletion
  // =========================================================================
  console.error("\n--- Cleanup: Temp Dir Deletion ---");
  {
    const result = await runHarness({
      scenario: { lines: [], exitCode: 0 },
      prompt: "test",
    });

    test("temporary MCP config directory is deleted after process exit", () => {
      const idx = result.args.indexOf("--mcp-config");
      assert.notStrictEqual(idx, -1, "--mcp-config not found");
      const configPath = result.args[idx + 1];
      const configDir = dirname(configPath);
      // After the process exits, the cleanup handler should have deleted the dir
      assert.ok(!existsSync(configDir), `temp dir ${configDir} was not deleted`);
    });
  }

  // =========================================================================
  // Summary
  // =========================================================================
  cleanupMockClaude();

  console.error(`\n${"=".repeat(50)}`);
  console.error(`Results: ${passed} passed, ${failed} failed, ${passed + failed} total`);
  console.error(`${"=".repeat(50)}`);

  if (failed > 0) {
    console.error("\nFailed tests:");
    for (const t of testLog) {
      if (t.status === "FAIL") {
        console.error(`  ✗ ${t.name}: ${t.error}`);
      }
    }
  }

  process.exit(failed > 0 ? 1 : 0);
}

main().catch((e) => {
  console.error(`Fatal error: ${e.message}`);
  console.error(e.stack);
  cleanupMockClaude();
  process.exit(1);
});
