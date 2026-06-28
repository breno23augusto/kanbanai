#!/usr/bin/env bash
set -a
. /home/breno/projects/kanbanai/.env.dev
set +a
cd /home/breno/projects/kanbanai

# The systemd unit runs with a minimal PATH (no nvm), so `npm root -g` inside
# pi-harness.sh would resolve to the system node_modules and fail to find the
# globally-installed @earendil-works/pi-coding-agent. Pin the nvm node bin dir
# and the package location explicitly.
NODE_BIN_DIR="$(dirname "$(readlink -f /home/breno/.nvm/versions/node/v24.16.0/bin/node)")"
export PATH="${NODE_BIN_DIR}:${PATH:-/usr/bin:/bin}"
export PI_PKG_DIR="/home/breno/.nvm/versions/node/v24.16.0/lib/node_modules/@earendil-works/pi-coding-agent"

exec /home/breno/projects/kanbanai/bin/kanbanai serve
