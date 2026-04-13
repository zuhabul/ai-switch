#!/usr/bin/env bash
set -euo pipefail

AISWITCH_BIN="${AISWITCH_BIN:-$HOME/.local/bin/aiswitch}"

if [[ ! -x "${AISWITCH_BIN}" ]]; then
  echo "aiswitch binary not found at ${AISWITCH_BIN}" >&2
  exit 1
fi

upsert_profile() {
  local id="$1" provider="$2" frontend="$3" auth="$4" protocol="$5" priority="$6" tags="$7" budget="$8"
  "${AISWITCH_BIN}" profile add \
    --id "${id}" \
    --provider "${provider}" \
    --frontend "${frontend}" \
    --auth "${auth}" \
    --protocol "${protocol}" \
    --priority "${priority}" \
    --tags "${tags}" \
    --budget "${budget}" \
    --enabled true >/dev/null
}

"${AISWITCH_BIN}" init >/dev/null || true

upsert_profile "codex-main" "openai" "codex" "chatgpt" "app_server" 10 "prod,primary" 20
upsert_profile "codex-backup" "openai" "codex" "chatgpt" "app_server" 8 "prod,backup" 8
upsert_profile "opencode-main" "openai" "opencode" "chatgpt" "native_cli" 7 "prod,primary" 8
upsert_profile "claude-main" "anthropic" "claude_code" "claude_app" "native_cli" 7 "prod,primary" 12
upsert_profile "gemini-main" "google" "gemini_cli" "google_login" "native_cli" 6 "prod,primary" 5
upsert_profile "hermes-main" "minimax" "hermes" "api_key" "hermes" 5 "prod" 5
upsert_profile "openclaw-main" "openclaw" "openclaw" "api_key" "native_cli" 4 "prod" 5

now_iso="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

set_health() {
  local id="$1" r5m="$2" rh="$3" lat="$4" err="$5"
  "${AISWITCH_BIN}" health set --id "${id}" --r5m "${r5m}" --rh "${rh}" --latency "${lat}" --error "${err}" >/dev/null
}

set_health "codex-main" 40 500 120 0.2
set_health "codex-backup" 30 350 180 0.4
set_health "opencode-main" 35 380 130 0.3
set_health "claude-main" 25 250 170 0.3
set_health "gemini-main" 25 300 130 0.2
set_health "hermes-main" 20 200 180 0.4
set_health "openclaw-main" 20 200 200 0.4

echo "Bootstrapped default profiles and health at ${now_iso}" >&2
