#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
AI_BIN="${ROOT_DIR}/ai"

pass() {
  echo "PASS: $*"
}

fail_case() {
  echo "FAIL: $*" >&2
  exit 1
}

create_fake_codex() {
  local fake_bin="$1"
  cat > "${fake_bin}" <<'EOF'
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
SESSION_FILE="${SESSION_DIR}/rollout-2026-01-01T00-00-00-session-xyz.jsonl"
mkdir -p "${SESSION_DIR}"

if [[ ! -f "${SESSION_FILE}" ]]; then
  cat > "${SESSION_FILE}" <<JSON
{"timestamp":"2026-01-01T00:00:00Z","type":"session_meta","payload":{"id":"session-xyz","cwd":"${PWD}"}}
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
  if [[ "${FAKE_NO_LIMIT:-0}" == "1" ]]; then
    echo "running with ${ACCOUNT_ID}"
    exit "${FAKE_FIRST_EXIT_CODE:-130}"
  fi
  LIMIT_MSG="${FAKE_LIMIT_MESSAGE:-}"
  if [[ -z "${LIMIT_MSG}" ]]; then
    LIMIT_MSG="You've hit your usage limit. Try again in 1 hour 0 minutes."
  fi
  SESSION_HINT_ID="${FAKE_SESSION_ID:-session-xyz}"
  echo "running with ${ACCOUNT_ID}"
  printf '%b\n' "${LIMIT_MSG}"
  echo "To continue this session, run codex resume ${SESSION_HINT_ID}"
  exit "${FAKE_FIRST_EXIT_CODE:-1}"
fi

if [[ -z "${RESUME_TARGET}" || "${RESUME_TARGET}" != "session-xyz" ]]; then
  echo "expected resume session-xyz, got: $*" >&2
  exit 2
fi

echo "resumed ${RESUME_TARGET} with ${ACCOUNT_ID}"
exit 0
EOF
  chmod +x "${fake_bin}"
}

seed_accounts() {
  cat > "${CODEX_AUTH_FILE}" <<'JSON'
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
JSON
  "${AI_BIN}" save acc1 >/dev/null

  cat > "${CODEX_AUTH_FILE}" <<'JSON'
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
JSON
  "${AI_BIN}" save acc2 >/dev/null
  "${AI_BIN}" pool-set acc1 acc2 >/dev/null
}

run_case() {
  local name="$1"
  local limit_msg="$2"
  local first_exit_code="$3"
  local expect_switch="$4"
  local expected_min_cooldown="$5"

  local test_root
  test_root="$(mktemp -d)"
  trap 'rm -rf "${test_root}"' RETURN

  export HOME="${test_root}/home"
  export CODEX_HOME="${HOME}/.codex"
  export AI_SWITCH_HOME="${HOME}/.local/share/ai-switch"
  export CODEX_AUTH_FILE="${CODEX_HOME}/auth.json"
  export AI_SWITCH_CODEX_BIN="${test_root}/bin/fake-codex"

  mkdir -p "${HOME}" "${CODEX_HOME}" "${test_root}/bin"
  create_fake_codex "${AI_SWITCH_CODEX_BIN}"
  seed_accounts

  local output
  output="$(
    FAKE_LIMIT_MESSAGE="${limit_msg}" \
    FAKE_FIRST_EXIT_CODE="${first_exit_code}" \
    "${AI_BIN}" codex --no-alt-screen 2>&1 || true
  )"

  if [[ "${expect_switch}" == "yes" ]]; then
    grep -q "Rate limit detected for 'acc1'. Switched to 'acc2' and resuming session 'session-xyz'." <<<"${output}" \
      || fail_case "${name}: expected switch output, got: ${output}"
    grep -q "resumed session-xyz with account-b" <<<"${output}" \
      || fail_case "${name}: expected resume on acc2, got: ${output}"

    local until now delta
    until="$(jq -r '.cooldowns.acc1.until_epoch // 0' "${AI_SWITCH_HOME}/state/codex-pool.json")"
    now="$(date +%s)"
    delta=$((until - now))
    (( delta >= expected_min_cooldown )) \
      || fail_case "${name}: cooldown too small (${delta}s < ${expected_min_cooldown}s)"
  else
    if grep -q "Rate limit detected for 'acc1'" <<<"${output}"; then
      fail_case "${name}: expected no switch, got: ${output}"
    fi
  fi

  pass "${name}"
}

run_pool_bootstrap_case() {
  local test_root
  test_root="$(mktemp -d)"
  trap 'rm -rf "${test_root}"' RETURN

  export HOME="${test_root}/home"
  export CODEX_HOME="${HOME}/.codex"
  export AI_SWITCH_HOME="${HOME}/.local/share/ai-switch"
  export CODEX_AUTH_FILE="${CODEX_HOME}/auth.json"
  export AI_SWITCH_CODEX_BIN="${test_root}/bin/fake-codex"

  mkdir -p "${HOME}" "${CODEX_HOME}" "${test_root}/bin"
  create_fake_codex "${AI_SWITCH_CODEX_BIN}"
  seed_accounts

  mkdir -p "${AI_SWITCH_HOME}/state"
  printf '%s\n' '{"accounts":[],"cooldowns":{},"last_account":null,"updated_at":null}' > "${AI_SWITCH_HOME}/state/codex-pool.json"

  local output
  output="$("${AI_BIN}" codex --no-alt-screen 2>&1 || true)"
  grep -q "Initialized Codex rotation pool from saved accounts." <<<"${output}" \
    || fail_case "pool-bootstrap: missing init message"
  [[ "$(jq '.accounts | length' "${AI_SWITCH_HOME}/state/codex-pool.json")" -ge 2 ]] \
    || fail_case "pool-bootstrap: pool did not repopulate"

  pass "pool-bootstrap"
}

run_all_accounts_unavailable_case() {
  local test_root
  test_root="$(mktemp -d)"
  trap 'rm -rf "${test_root}"' RETURN

  export HOME="${test_root}/home"
  export CODEX_HOME="${HOME}/.codex"
  export AI_SWITCH_HOME="${HOME}/.local/share/ai-switch"
  export CODEX_AUTH_FILE="${CODEX_HOME}/auth.json"
  export AI_SWITCH_CODEX_BIN="${test_root}/bin/fake-codex"

  mkdir -p "${HOME}" "${CODEX_HOME}" "${test_root}/bin"
  create_fake_codex "${AI_SWITCH_CODEX_BIN}"
  seed_accounts

  "${AI_BIN}" pool-cooldown acc2 86400 >/dev/null
  local output
  output="$(
    FAKE_LIMIT_MESSAGE="You've hit your usage limit. Try again in 1 hour 0 minutes." \
    FAKE_FIRST_EXIT_CODE=1 \
    "${AI_BIN}" codex --no-alt-screen 2>&1 || true
  )"

  grep -q "no other pool account is currently available" <<<"${output}" \
    || fail_case "all-accounts-unavailable: expected hard-fail message, got: ${output}"

  pass "all-accounts-unavailable"
}

run_resume_from_log_only_case() {
  local test_root
  test_root="$(mktemp -d)"
  trap 'rm -rf "${test_root}"' RETURN

  export HOME="${test_root}/home"
  export CODEX_HOME="${HOME}/.codex"
  export AI_SWITCH_HOME="${HOME}/.local/share/ai-switch"
  export CODEX_AUTH_FILE="${CODEX_HOME}/auth.json"
  export AI_SWITCH_CODEX_BIN="${test_root}/bin/fake-codex"

  mkdir -p "${HOME}" "${CODEX_HOME}" "${test_root}/bin"
  create_fake_codex "${AI_SWITCH_CODEX_BIN}"
  seed_accounts

  # Remove any session files so fallback must come from log output.
  rm -rf "${CODEX_HOME}/sessions"
  local output
  output="$(
    FAKE_FIRST_EXIT_CODE=1 \
    FAKE_LIMIT_MESSAGE="You've hit your usage limit. Try again in 1 hour 0 minutes." \
    FAKE_SESSION_ID="session-xyz" \
    "${AI_BIN}" codex --no-alt-screen 2>&1 || true
  )"

  grep -q "Rate limit detected for 'acc1'. Switched to 'acc2' and resuming session 'session-xyz'." <<<"${output}" \
    || fail_case "resume-from-log-only: expected fallback resume via log, got: ${output}"
  grep -q "resumed session-xyz with account-b" <<<"${output}" \
    || fail_case "resume-from-log-only: expected resumed execution, got: ${output}"

  pass "resume-from-log-only"
}

run_sigint_resume_persistence_case() {
  local test_root
  test_root="$(mktemp -d)"
  trap 'rm -rf "${test_root}"' RETURN

  export HOME="${test_root}/home"
  export CODEX_HOME="${HOME}/.codex"
  export AI_SWITCH_HOME="${HOME}/.local/share/ai-switch"
  export CODEX_AUTH_FILE="${CODEX_HOME}/auth.json"
  export AI_SWITCH_CODEX_BIN="${test_root}/bin/fake-codex"

  mkdir -p "${HOME}" "${CODEX_HOME}" "${test_root}/bin"
  create_fake_codex "${AI_SWITCH_CODEX_BIN}"
  seed_accounts

  local first_output
  local first_status
  set +e
  first_output="$(
    FAKE_NO_LIMIT=1 \
    FAKE_FIRST_EXIT_CODE=130 \
    "${AI_BIN}" codex --no-alt-screen 2>&1
  )"
  first_status=$?
  set -e

  [[ "${first_status}" -eq 130 ]] \
    || fail_case "sigint-resume-persistence: expected first exit 130, got ${first_status} :: ${first_output}"
  [[ "$(jq -r '.pending_resume // false' "${AI_SWITCH_HOME}/state/codex-session.json")" == "true" ]] \
    || fail_case "sigint-resume-persistence: pending_resume was not preserved"
  [[ "$(jq -r '.session_id // empty' "${AI_SWITCH_HOME}/state/codex-session.json")" == "session-xyz" ]] \
    || fail_case "sigint-resume-persistence: missing session-xyz state"

  local output
  output="$("${AI_BIN}" codex --no-alt-screen 2>&1 || true)"
  grep -q "Resuming interrupted managed Codex session 'session-xyz'." <<<"${output}" \
    || fail_case "sigint-resume-persistence: missing interrupted-session resume notice"
  grep -q "Starting managed Codex with account 'acc1'." <<<"${output}" \
    || fail_case "sigint-resume-persistence: expected preferred account acc1, got: ${output}"
  grep -q "resumed session-xyz with account-a" <<<"${output}" \
    || fail_case "sigint-resume-persistence: expected resume on original account, got: ${output}"
  [[ "$(jq -r '.pending_resume // false' "${AI_SWITCH_HOME}/state/codex-session.json")" == "false" ]] \
    || fail_case "sigint-resume-persistence: pending_resume should clear after clean resume"

  pass "sigint-resume-persistence"
}

run_ignore_foreign_newer_session_case() {
  local test_root
  test_root="$(mktemp -d)"
  trap 'rm -rf "${test_root}"' RETURN

  export HOME="${test_root}/home"
  export CODEX_HOME="${HOME}/.codex"
  export AI_SWITCH_HOME="${HOME}/.local/share/ai-switch"
  export CODEX_AUTH_FILE="${CODEX_HOME}/auth.json"
  export AI_SWITCH_CODEX_BIN="${test_root}/bin/fake-codex"

  mkdir -p "${HOME}" "${CODEX_HOME}" "${test_root}/bin"
  create_fake_codex "${AI_SWITCH_CODEX_BIN}"
  seed_accounts

  local foreign_dir="${CODEX_HOME}/sessions/2099/01/01"
  local foreign_file="${foreign_dir}/foreign-session.jsonl"
  mkdir -p "${foreign_dir}"
  cat > "${foreign_file}" <<'JSON'
{"timestamp":"2099-01-01T00:00:00Z","type":"session_meta","payload":{"id":"session-foreign","cwd":"/tmp/foreign-project"}}
JSON
  touch -d '+2 days' "${foreign_file}"

  local output
  output="$(
    FAKE_LIMIT_MESSAGE="You've hit your usage limit. Try again in 1 hour 0 minutes." \
    FAKE_FIRST_EXIT_CODE=1 \
    "${AI_BIN}" codex --no-alt-screen 2>&1 || true
  )"

  grep -q "Rate limit detected for 'acc1'. Switched to 'acc2' and resuming session 'session-xyz'." <<<"${output}" \
    || fail_case "ignore-foreign-newer-session: wrapper picked wrong session :: ${output}"
  grep -q "resumed session-xyz with account-b" <<<"${output}" \
    || fail_case "ignore-foreign-newer-session: expected correct session resume :: ${output}"

  pass "ignore-foreign-newer-session"
}

run_sigint_auto_rotate_same_chat_case() {
  local test_root
  test_root="$(mktemp -d)"
  trap 'rm -rf "${test_root}"' RETURN

  export HOME="${test_root}/home"
  export CODEX_HOME="${HOME}/.codex"
  export AI_SWITCH_HOME="${HOME}/.local/share/ai-switch"
  export CODEX_AUTH_FILE="${CODEX_HOME}/auth.json"
  export AI_SWITCH_CODEX_BIN="${test_root}/bin/fake-codex"

  mkdir -p "${HOME}" "${CODEX_HOME}" "${test_root}/bin" "${AI_SWITCH_HOME}/state"
  create_fake_codex "${AI_SWITCH_CODEX_BIN}"
  seed_accounts

  cat > "${AI_SWITCH_HOME}/state/codex-limits.json" <<'JSON'
{"accounts":{"acc1":{"primary":{"used_percent":95}}},"updated_at":"2026-01-01T00:00:00Z"}
JSON

  local output
  output="$(
    FAKE_NO_LIMIT=1 \
    FAKE_FIRST_EXIT_CODE=130 \
    "${AI_BIN}" codex --no-alt-screen 2>&1 || true
  )"

  grep -q "Ctrl+C detected during high usage" <<<"${output}" \
    || fail_case "sigint-auto-rotate-same-chat: missing high-usage interrupt detection :: ${output}"
  grep -q "Rate limit detected for 'acc1'. Switched to 'acc2' and resuming session 'session-xyz'." <<<"${output}" \
    || fail_case "sigint-auto-rotate-same-chat: did not reopen same session on acc2 :: ${output}"
  grep -q "resumed session-xyz with account-b" <<<"${output}" \
    || fail_case "sigint-auto-rotate-same-chat: resumed wrong chat/account :: ${output}"

  pass "sigint-auto-rotate-same-chat"
}

run_case "relative-in-1h" "You've hit your usage limit. Try again in 1 hour 0 minutes." 1 yes 3500

ABS_ORD="$(date -d '+2 hours' '+%b %-dth, %Y %-I:%M %p')"
run_case "absolute-at-ordinal" "You've hit your usage limit. Try again at ${ABS_ORD}." 1 yes 3600

run_case "http-429" "Error: 429 Too Many Requests" 1 yes 700

ANSI_MSG=$'\e[31mYou\'ve hit your usage limit.\e[0m Try again in 1 hour 0 minutes.'
run_case "ansi-polluted" "${ANSI_MSG}" 1 yes 3500

run_case "zero-exit-limited" "You've hit your usage limit. Try again in 1 hour 0 minutes." 0 yes 3500

run_case "non-limit-error" "network disconnected unexpectedly" 1 no 0

ABS_ONLY="$(date -d '+2 hours' '+%b %-d, %Y %-I:%M %p')"
run_case "absolute-at-only" "Try again at ${ABS_ONLY}." 1 yes 3600

run_pool_bootstrap_case
run_all_accounts_unavailable_case
run_resume_from_log_only_case
run_sigint_resume_persistence_case
run_ignore_foreign_newer_session_case
run_sigint_auto_rotate_same_chat_case

echo "ALL MATRIX TESTS PASSED"
