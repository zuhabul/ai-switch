#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
AI_BIN="${ROOT_DIR}/ai"
TEST_ROOT="$(mktemp -d)"
trap 'rm -rf "${TEST_ROOT}"' EXIT

export HOME="${TEST_ROOT}/home"
export CODEX_HOME="${HOME}/.codex"
export AI_SWITCH_HOME="${HOME}/.local/share/ai-switch"
export CODEX_AUTH_FILE="${CODEX_HOME}/auth.json"
export AI_SWITCH_CODEX_BIN="${TEST_ROOT}/bin/fake-codex"

mkdir -p "${HOME}" "${CODEX_HOME}" "${TEST_ROOT}/bin"

cat > "${AI_SWITCH_CODEX_BIN}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

STATE_FILE="${HOME}/fake-codex-state"
COUNT=0
if [[ -f "${STATE_FILE}" ]]; then
  COUNT="$(cat "${STATE_FILE}")"
fi
COUNT=$((COUNT + 1))
echo "${COUNT}" > "${STATE_FILE}"

SESSION_DIR="${CODEX_HOME}/sessions/2026/01/01"
SESSION_FILE="${SESSION_DIR}/rollout-2026-01-01T00-00-00-session-123.jsonl"
mkdir -p "${SESSION_DIR}"

if [[ ! -f "${SESSION_FILE}" ]]; then
  cat > "${SESSION_FILE}" <<JSON
{"timestamp":"2026-01-01T00:00:00Z","type":"session_meta","payload":{"id":"session-123","cwd":"${PWD}"}}
JSON
fi

ACCOUNT_ID="$(jq -r '.tokens.account_id' "${CODEX_AUTH_FILE}")"
RESUME_TARGET=""
for ((i = 1; i <= $#; i++)); do
  if [[ "${!i}" == "resume" ]]; then
    next_index=$((i + 1))
    RESUME_TARGET="${!next_index:-}"
    break
  fi
done

if [[ "${COUNT}" -eq 1 ]]; then
  LIMIT_MSG="${FAKE_LIMIT_MESSAGE:-}"
  if [[ -z "${LIMIT_MSG}" ]]; then
    LIMIT_MSG="You've hit your usage limit. Try again in 1 hour 0 minutes."
  fi
  echo "running with ${ACCOUNT_ID}"
  echo "${LIMIT_MSG}"
  exit "${FAKE_FIRST_EXIT_CODE:-1}"
fi

if [[ -z "${RESUME_TARGET}" || "${RESUME_TARGET}" != "session-123" ]]; then
  echo "expected resume session-123, got: $*" >&2
  exit 2
fi

echo "resumed ${RESUME_TARGET} with ${ACCOUNT_ID}"
exit 0
EOF
chmod +x "${AI_SWITCH_CODEX_BIN}"

cat > "${CODEX_AUTH_FILE}" <<'EOF'
{
  "OPENAI_API_KEY": null,
  "auth_mode": "chatgpt",
  "last_refresh": "2026-01-01T00:00:00Z",
  "tokens": {
    "access_token": "token-a",
    "refresh_token": "refresh-a",
    "account_id": "account-a",
    "id_token": null
  }
}
EOF

"${AI_BIN}" save acc1 >/dev/null

cat > "${CODEX_AUTH_FILE}" <<'EOF'
{
  "OPENAI_API_KEY": null,
  "auth_mode": "chatgpt",
  "last_refresh": "2026-01-01T00:00:00Z",
  "tokens": {
    "access_token": "token-b",
    "refresh_token": "refresh-b",
    "account_id": "account-b",
    "id_token": null
  }
}
EOF

"${AI_BIN}" save acc2 >/dev/null
"${AI_BIN}" pool-set acc1 acc2 >/dev/null

OUTPUT="$("${AI_BIN}" codex --no-alt-screen 2>&1)"
echo "${OUTPUT}"

grep -q "Starting managed Codex with account 'acc1'." <<<"${OUTPUT}"
grep -q "Rate limit detected for 'acc1'. Switched to 'acc2' and resuming session 'session-123'." <<<"${OUTPUT}"
grep -q "resumed session-123 with account-b" <<<"${OUTPUT}"

COOLDOWN_FILE="${AI_SWITCH_HOME}/state/codex-pool.json"
ACTIVE_UNTIL="$(jq -r '.cooldowns.acc1.until_epoch // 0' "${COOLDOWN_FILE}")"
if [[ "${ACTIVE_UNTIL}" -le 0 ]]; then
  echo "expected acc1 cooldown to be recorded" >&2
  exit 3
fi

rm -f "${HOME}/fake-codex-state"
"${AI_BIN}" pool-reset-cooldowns >/dev/null

OUTPUT="$(FAKE_FIRST_EXIT_CODE=0 "${AI_BIN}" codex resume session-123 --no-alt-screen 2>&1)"
echo "${OUTPUT}"

grep -q "Starting managed Codex with account 'acc1'." <<<"${OUTPUT}"
grep -q "Rate limit detected for 'acc1'. Switched to 'acc2' and resuming session 'session-123'." <<<"${OUTPUT}"
grep -q "resumed session-123 with account-b" <<<"${OUTPUT}"

ACTIVE_UNTIL="$(jq -r '.cooldowns.acc1.until_epoch // 0' "${COOLDOWN_FILE}")"
if [[ "${ACTIVE_UNTIL}" -le 0 ]]; then
  echo "expected acc1 cooldown to be recorded after zero-exit rate limit" >&2
  exit 4
fi

rm -f "${HOME}/fake-codex-state"
"${AI_BIN}" pool-reset-cooldowns >/dev/null

ABSOLUTE_RETRY_TEXT="$(date -d '+2 hours' '+%b %-d, %Y %-I:%M %p')"
OUTPUT="$(
  FAKE_FIRST_EXIT_CODE=0 \
  FAKE_LIMIT_MESSAGE="You've hit your usage limit. Try again at ${ABSOLUTE_RETRY_TEXT}." \
  "${AI_BIN}" codex resume session-123 --no-alt-screen 2>&1
)"
echo "${OUTPUT}"

grep -q "Rate limit detected for 'acc1'. Switched to 'acc2' and resuming session 'session-123'." <<<"${OUTPUT}"

ACTIVE_UNTIL="$(jq -r '.cooldowns.acc1.until_epoch // 0' "${COOLDOWN_FILE}")"
EXPECTED_MIN=$(( $(date +%s) + 60 * 60 ))
if [[ "${ACTIVE_UNTIL}" -lt "${EXPECTED_MIN}" ]]; then
  echo "expected acc1 cooldown to reflect parsed absolute retry time (>=1 hour)" >&2
  exit 5
fi
