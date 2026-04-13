#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
AI_BIN="${ROOT_DIR}/ai"

fail_case() {
  echo "FAIL: $*" >&2
  exit 1
}

pass_case() {
  echo "PASS: $*"
}

TEST_ROOT="$(mktemp -d)"
trap 'rm -rf "${TEST_ROOT}"' EXIT

export HOME="${TEST_ROOT}/home"
export CODEX_HOME="${HOME}/.codex"
export AI_SWITCH_HOME="${HOME}/.local/share/ai-switch"
export CODEX_AUTH_FILE="${CODEX_HOME}/auth.json"

mkdir -p "${CODEX_HOME}" "${AI_SWITCH_HOME}/credentials" "${AI_SWITCH_HOME}/state"

cat > "${AI_SWITCH_HOME}/credentials/acc1.json" <<'JSON'
{
  "format": "unified-openai-auth",
  "version": 1,
  "tokens": {
    "access_token": "a1",
    "refresh_token": "r1",
    "account_id": "account-a",
    "id_token": null
  },
  "codex": {
    "auth_mode": "chatgpt",
    "openai_api_key": null,
    "last_refresh": "2026-04-07T18:00:00Z"
  },
  "opencode": {
    "openai": {
      "access": "a1",
      "refresh": "r1",
      "accountId": "account-a",
      "expires": 0,
      "type": "oauth"
    }
  },
  "meta": {
    "updated_at": "2026-04-07T18:00:00Z"
  }
}
JSON

cat > "${AI_SWITCH_HOME}/credentials/acc2.json" <<'JSON'
{
  "format": "unified-openai-auth",
  "version": 1,
  "tokens": {
    "access_token": "a2",
    "refresh_token": "r2",
    "account_id": "account-b",
    "id_token": null
  },
  "codex": {
    "auth_mode": "chatgpt",
    "openai_api_key": null,
    "last_refresh": "2026-04-07T18:00:00Z"
  },
  "opencode": {
    "openai": {
      "access": "a2",
      "refresh": "r2",
      "accountId": "account-b",
      "expires": 0,
      "type": "oauth"
    }
  },
  "meta": {
    "updated_at": "2026-04-07T18:00:00Z"
  }
}
JSON

cat > "${CODEX_AUTH_FILE}" <<'JSON'
{
  "OPENAI_API_KEY": null,
  "auth_mode": "chatgpt",
  "last_refresh": "2026-04-07T18:00:00Z",
  "tokens": {
    "access_token": "active-a",
    "refresh_token": "active-r",
    "account_id": "account-a",
    "id_token": null
  }
}
JSON

NOW="$(date +%s)"
COOLDOWN_UNTIL="$((NOW + 1800))"
PRIMARY_RESET="$((NOW + 7200))"
SECONDARY_RESET="$((NOW + 172800))"

cat > "${AI_SWITCH_HOME}/state/codex-pool.json" <<JSON
{
  "accounts": ["acc1", "acc2"],
  "cooldowns": {
    "acc2": {
      "until_epoch": ${COOLDOWN_UNTIL},
      "reason": "rate_limited",
      "updated_at": "2026-04-07T18:00:00Z"
    }
  },
  "last_account": "acc1",
  "updated_at": "2026-04-07T18:00:00Z"
}
JSON

cat > "${AI_SWITCH_HOME}/state/codex-limits.json" <<JSON
{
  "accounts": {
    "acc1": {
      "captured_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
      "source": "chatgpt-session",
      "session_id": "s-1",
      "limit_id": "codex",
      "plan_type": "plus",
      "primary": {
        "used_percent": 12.0,
        "window_minutes": 300,
        "resets_at": ${PRIMARY_RESET}
      },
      "secondary": {
        "used_percent": 45.0,
        "window_minutes": 10080,
        "resets_at": ${SECONDARY_RESET}
      },
      "credits": null
    },
    "acc2": {
      "captured_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
      "source": "chatgpt-session",
      "session_id": "s-2",
      "limit_id": "codex",
      "plan_type": "plus",
      "primary": {
        "used_percent": 86.0,
        "window_minutes": 300,
        "resets_at": ${PRIMARY_RESET}
      },
      "secondary": {
        "used_percent": 95.0,
        "window_minutes": 10080,
        "resets_at": ${SECONDARY_RESET}
      },
      "credits": null
    }
  },
  "updated_at": "2026-04-07T18:00:00Z"
}
JSON

status_out="$("${AI_BIN}" status 2>&1)"
grep -q "AI STATUS " <<< "${status_out}" || fail_case "status header missing"
grep -q "AR acc1" <<< "${status_out}" || fail_case "active account compact row missing"
grep -q -- "-C acc2" <<< "${status_out}" || fail_case "cooling account compact row missing"
grep -q "ST ACCOUNT" <<< "${status_out}" || fail_case "status table header missing"
grep -q "\[##" <<< "${status_out}" || fail_case "compact usage bars missing"
pass_case "status-ui"

usage_out="$("${AI_BIN}" usage 2>&1)"
grep -q "5h 12%" <<< "${usage_out}" || fail_case "usage 5h summary missing"
grep -q "weekly 45%" <<< "${usage_out}" || fail_case "usage weekly summary missing"
pass_case "usage-summary"

pool_out="$("${AI_BIN}" pool-status 2>&1)"
grep -q "acc2: active, reason=rate_limited" <<< "${pool_out}" || fail_case "pool-status human cooldown missing"
pass_case "pool-status-readable"

cat > "${CODEX_AUTH_FILE}" <<'JSON'
{
  "OPENAI_API_KEY": null,
  "auth_mode": "chatgpt",
  "last_refresh": "2026-04-07T18:00:00Z",
  "tokens": {
    "access_token": "active-b",
    "refresh_token": "active-rb",
    "account_id": "account-b",
    "id_token": null
  }
}
JSON

cat > "${AI_SWITCH_HOME}/state/codex-pool.json" <<'JSON'
{
  "accounts": ["acc1", "ghost"],
  "cooldowns": {
    "ghost": {
      "until_epoch": 9999999999,
      "reason": "rate_limited",
      "updated_at": "2026-04-07T18:00:00Z"
    }
  },
  "last_account": "ghost",
  "updated_at": "2026-04-07T18:00:00Z"
}
JSON

rotate_out="$("${AI_BIN}" rotate --no-refresh 2>&1)"
grep -q "Reconciled Codex rotation pool with saved accounts." <<< "${rotate_out}" || fail_case "rotate did not auto reconcile pool"
grep -q "Rotated Codex account: acc2 -> acc1" <<< "${rotate_out}" || fail_case "rotate did not switch to healthiest ready account"
grep -q '"account_id": "account-a"' "${CODEX_AUTH_FILE}" || fail_case "rotate did not update active codex auth"
jq -e '.accounts | index("acc2") != null' "${AI_SWITCH_HOME}/state/codex-pool.json" >/dev/null || fail_case "reconciled pool missing saved account acc2"
jq -e '.accounts | index("ghost") == null' "${AI_SWITCH_HOME}/state/codex-pool.json" >/dev/null || fail_case "reconciled pool did not remove stale account"
pass_case "rotate-healthiest"

echo "STATUS UI TESTS PASSED"
