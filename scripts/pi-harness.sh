#!/usr/bin/env bash
#
# pi-harness.sh — KanbanAI harness launcher for the pi coding agent (SDK-based).
#
# KanbanAI's CommandBuilder invokes the harness as: <cmd> --model <model> --prompt <prompt>
# with KANBANAI_TASK_ID / KANBANAI_MCP_PORT / KANBANAI_MCP_URL in the env. This
# launcher locates the globally-installed @earendil-works/pi-coding-agent package
# and runs the SDK harness script, forwarding the args unchanged.
#
# The actual agent execution + phase-completion bridging lives in pi-harness.mjs.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Resolve the global node_modules root (works under nvm too).
GLOBAL_NM="$(npm root -g 2>/dev/null || true)"
PKG_DIR="${PI_PKG_DIR:-${GLOBAL_NM}/@earendil-works/pi-coding-agent}"

if [[ ! -d "$PKG_DIR/dist" ]]; then
  echo "pi-harness: pi SDK package not found at $PKG_DIR" >&2
  echo "  set PI_PKG_DIR or ensure @earendil-works/pi-coding-agent is installed (npm i -g)" >&2
  exit 2
fi

exec env PI_PKG_DIR="$PKG_DIR" node "${SCRIPT_DIR}/pi-harness.mjs" "$@"