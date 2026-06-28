#!/usr/bin/env bash
#
# claude-harness.sh — KanbanAI harness launcher for Claude Code (headless).
#
# KanbanAI's CommandBuilder invokes the harness as: <cmd> --model <model> --prompt <prompt>
# with KANBANAI_TASK_ID / KANBANAI_PHASE / KANBANAI_MCP_URL / KANBANAI_API_BASE_URL
# in the env. This launcher runs the Node harness script that spawns `claude -p`
# with the KanbanAI MCP server attached and pretty-prints stream-json to stderr
# for the live tail.
#
# Claude Code speaks MCP natively, so — unlike pi-harness.sh — no customTools/REST
# shim is needed: the agent calls the real KanbanAI MCP tools directly.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Resolve node from nvm if not already on PATH (the systemd service sets PATH in
# scripts/run-kanbanai.sh, but this also works for manual runs).
if ! command -v node >/dev/null 2>&1; then
  if [[ -n "${NVM_DIR:-}" && -s "$NVM_DIR/nvm.sh" ]]; then
    # shellcheck disable=SC1091
    . "$NVM_DIR/nvm.sh"
  fi
fi

exec node "${SCRIPT_DIR}/claude-harness.mjs" "$@"