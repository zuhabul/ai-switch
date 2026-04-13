#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
V2_DIR="${ROOT_DIR}/v2"

require_cmd() {
  local cmd="$1"
  command -v "${cmd}" >/dev/null 2>&1 || {
    echo "FAIL: required command not found: ${cmd}" >&2
    exit 1
  }
}

pass_case() {
  echo "PASS: $*"
}

fail_case() {
  echo "FAIL: $*" >&2
  exit 1
}

require_cmd go
require_cmd curl
require_cmd python3

TMP_ROOT="$(mktemp -d)"
trap '[[ -n "${DAEMON_PID:-}" ]] && kill "${DAEMON_PID}" >/dev/null 2>&1 || true; rm -rf "${TMP_ROOT}"' EXIT

# Keep Go caches outside test HOME to avoid cleanup permission issues from toolchain cache.
export GOCACHE="${GOCACHE:-/tmp/ai-switch-gocache}"
export GOMODCACHE="${GOMODCACHE:-/tmp/ai-switch-gomodcache}"
mkdir -p "${GOCACHE}" "${GOMODCACHE}"

BIN_DIR="${TMP_ROOT}/bin"
mkdir -p "${BIN_DIR}"

HOME_DIR="${TMP_ROOT}/home"
export HOME="${HOME_DIR}"
mkdir -p "${HOME}"

AISWITCH_BIN="${BIN_DIR}/aiswitch"
AISWITCHD_BIN="${BIN_DIR}/aiswitchd"

go -C "${V2_DIR}" build -o "${AISWITCH_BIN}" ./cmd/aiswitch
go -C "${V2_DIR}" build -o "${AISWITCHD_BIN}" ./cmd/aiswitchd

"${AISWITCH_BIN}" init >/dev/null

"${AISWITCH_BIN}" profile add --id codex-main --provider openai --frontend codex --auth chatgpt --protocol app_server --priority 10 --tags prod,primary --budget 20 >/dev/null
"${AISWITCH_BIN}" profile add --id codex-backup --provider openai --frontend codex --auth chatgpt --protocol app_server --priority 7 --tags prod,backup --budget 8 >/dev/null
"${AISWITCH_BIN}" profile add --id claude-ops --provider anthropic --frontend claude_code --auth api_key --protocol native_cli --priority 6 --tags prod,ops --budget 10 >/dev/null
"${AISWITCH_BIN}" profile add --id grok-exp --provider xai --frontend grok --auth api_key --protocol openai_compatible --priority 3 --tags exp --budget 2 >/dev/null

"${AISWITCH_BIN}" policy add --name deny-xai-coding --priority 200 --tasks coding --deny-providers xai >/dev/null
"${AISWITCH_BIN}" policy add --name codex-prod-only --priority 150 --frontends codex --require-any-tag prod >/dev/null

"${AISWITCH_BIN}" health set --id codex-main --r5m 40 --rh 400 --latency 120 --error 0.1 >/dev/null
"${AISWITCH_BIN}" health set --id codex-backup --r5m 30 --rh 300 --latency 180 --error 0.2 >/dev/null
"${AISWITCH_BIN}" health set --id claude-ops --r5m 20 --rh 200 --latency 160 --error 0.2 >/dev/null
"${AISWITCH_BIN}" health set --id grok-exp --r5m 80 --rh 600 --latency 80 --error 0.1 >/dev/null

"${AISWITCH_BIN}" secret set --name openai_back_key --value sk-live-backup >/dev/null
"${AISWITCH_BIN}" secret bind --profile codex-backup --env OPENAI_API_KEY --name openai_back_key >/dev/null

ROUTE_MAIN_JSON="${TMP_ROOT}/route_main.json"
"${AISWITCH_BIN}" route --frontend codex --task coding --protocol app_server > "${ROUTE_MAIN_JSON}"
python3 - "${ROUTE_MAIN_JSON}" <<'PY'
import json,sys
obj=json.load(open(sys.argv[1]))
if obj.get("profile_id") != "codex-main":
    raise SystemExit(f"expected codex-main, got {obj.get('profile_id')}")
PY
pass_case "route-selects-primary"

"${AISWITCH_BIN}" profile cooldown --id codex-main --for 30m >/dev/null
ROUTE_FAILOVER_JSON="${TMP_ROOT}/route_failover.json"
"${AISWITCH_BIN}" route --frontend codex --task coding --protocol app_server > "${ROUTE_FAILOVER_JSON}"
python3 - "${ROUTE_FAILOVER_JSON}" <<'PY'
import json,sys
obj=json.load(open(sys.argv[1]))
if obj.get("profile_id") != "codex-backup":
    raise SystemExit(f"expected codex-backup, got {obj.get('profile_id')}")
PY
pass_case "route-fails-over-on-cooldown"

LEASE_A_JSON="${TMP_ROOT}/lease_a.json"
"${AISWITCH_BIN}" lease acquire --profile codex-backup --frontend codex --owner e2e --ttl 10m > "${LEASE_A_JSON}"
python3 - "${LEASE_A_JSON}" <<'PY'
import json,sys
obj=json.load(open(sys.argv[1]))
if not obj.get("id"):
    raise SystemExit("lease id missing")
PY
pass_case "lease-acquire"

set +e
"${AISWITCH_BIN}" lease acquire --profile codex-backup --frontend codex --owner second --ttl 10m >/tmp/v2_lease_conflict.out 2>/tmp/v2_lease_conflict.err
lease_conflict_code=$?
set -e
if [[ "${lease_conflict_code}" -eq 0 ]]; then
  fail_case "lease collision expected failure but succeeded"
fi
pass_case "lease-lock-enforced"

LEASE_ID="$(python3 - "${LEASE_A_JSON}" <<'PY'
import json,sys
print(json.load(open(sys.argv[1]))["id"])
PY
)"
"${AISWITCH_BIN}" lease release --id "${LEASE_ID}" >/dev/null
pass_case "lease-release"

RUNTIME_PLAN_JSON="${TMP_ROOT}/runtime_plan.json"
"${AISWITCH_BIN}" runtime plan --frontend codex --task coding --protocol app_server --owner runtime-e2e > "${RUNTIME_PLAN_JSON}"
python3 - "${RUNTIME_PLAN_JSON}" <<'PY'
import json,sys
obj=json.load(open(sys.argv[1]))
if obj.get("profile_id") != "codex-backup":
    raise SystemExit(f"expected codex-backup runtime plan, got {obj.get('profile_id')}")
env=obj.get("env", {})
if env.get("OPENAI_API_KEY") != "sk-live-backup":
    raise SystemExit("runtime plan did not inject bound secret env")
if not obj.get("lease_id"):
    raise SystemExit("runtime plan did not include lease id")
PY
RUNTIME_LEASE_ID="$(python3 - "${RUNTIME_PLAN_JSON}" <<'PY'
import json,sys
print(json.load(open(sys.argv[1]))["lease_id"])
PY
)"
"${AISWITCH_BIN}" runtime release --lease "${RUNTIME_LEASE_ID}" >/dev/null
pass_case "runtime-plan-cli"

PORT=4467
"${AISWITCHD_BIN}" --addr "127.0.0.1:${PORT}" >"${TMP_ROOT}/aiswitchd.log" 2>&1 &
DAEMON_PID=$!

for _ in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:${PORT}/healthz" >"${TMP_ROOT}/healthz.json" 2>/dev/null; then
    break
  fi
  sleep 0.2
done
curl -fsS "http://127.0.0.1:${PORT}/healthz" >"${TMP_ROOT}/healthz.json"
python3 - "${TMP_ROOT}/healthz.json" <<'PY'
import json,sys
obj=json.load(open(sys.argv[1]))
if obj.get("ok") is not True:
    raise SystemExit("healthz did not return ok=true")
PY
pass_case "daemon-healthz"

curl -fsS -X POST "http://127.0.0.1:${PORT}/v2/route" \
  -H 'content-type: application/json' \
  -d '{"frontend":"codex","task_class":"coding","required_protocol":"app_server"}' > "${TMP_ROOT}/route_api.json"
python3 - "${TMP_ROOT}/route_api.json" <<'PY'
import json,sys
obj=json.load(open(sys.argv[1]))
if obj.get("profile_id") != "codex-backup":
    raise SystemExit(f"expected codex-backup via API, got {obj.get('profile_id')}")
PY
pass_case "daemon-route-api"

curl -fsS "http://127.0.0.1:${PORT}/v2/profiles" > "${TMP_ROOT}/profiles_api.json"
python3 - "${TMP_ROOT}/profiles_api.json" <<'PY'
import json,sys
arr=json.load(open(sys.argv[1]))
ids={x.get("id") for x in arr}
required={"codex-main","codex-backup","claude-ops","grok-exp"}
if not required.issubset(ids):
    raise SystemExit(f"missing profiles: {sorted(required-ids)}")
PY
pass_case "daemon-profiles-api"

curl -fsS -X POST "http://127.0.0.1:${PORT}/v2/runtime/plan" \
  -H 'content-type: application/json' \
  -d '{"frontend":"codex","task_class":"coding","required_protocol":"app_server","owner":"api-runtime-e2e"}' > "${TMP_ROOT}/runtime_plan_api.json"
python3 - "${TMP_ROOT}/runtime_plan_api.json" <<'PY'
import json,sys
obj=json.load(open(sys.argv[1]))
if obj.get("profile_id") != "codex-backup":
    raise SystemExit(f"expected codex-backup from runtime plan API, got {obj.get('profile_id')}")
if not obj.get("lease_id"):
    raise SystemExit("runtime plan API lease missing")
if obj.get("env", {}).get("OPENAI_API_KEY") != "sk-live-backup":
    raise SystemExit("runtime plan API missing bound secret env")
PY
RUNTIME_API_LEASE_ID="$(python3 - "${TMP_ROOT}/runtime_plan_api.json" <<'PY'
import json,sys
print(json.load(open(sys.argv[1]))["lease_id"])
PY
)"
curl -fsS -X DELETE "http://127.0.0.1:${PORT}/v2/leases?lease_id=${RUNTIME_API_LEASE_ID}" > /dev/null
pass_case "daemon-runtime-plan-api"

curl -fsS -X POST "http://127.0.0.1:${PORT}/v2/leases" \
  -H 'content-type: application/json' \
  -d '{"profile_id":"codex-main","frontend":"codex","owner":"api-e2e","ttl_min":5}' > "${TMP_ROOT}/lease_api.json"
python3 - "${TMP_ROOT}/lease_api.json" <<'PY'
import json,sys
obj=json.load(open(sys.argv[1]))
if not obj.get("id"):
    raise SystemExit("lease API did not return id")
PY
pass_case "daemon-lease-api-acquire"

LEASE_API_ID="$(python3 - "${TMP_ROOT}/lease_api.json" <<'PY'
import json,sys
print(json.load(open(sys.argv[1]))["id"])
PY
)"

curl -fsS "http://127.0.0.1:${PORT}/v2/leases" > "${TMP_ROOT}/leases_api.json"
python3 - "${TMP_ROOT}/leases_api.json" "${LEASE_API_ID}" <<'PY'
import json,sys
arr=json.load(open(sys.argv[1]))
needle=sys.argv[2]
if not any(x.get("id") == needle for x in arr):
    raise SystemExit("lease id not found in list")
PY
pass_case "daemon-lease-api-list"

curl -fsS -X DELETE "http://127.0.0.1:${PORT}/v2/leases?lease_id=${LEASE_API_ID}" > "${TMP_ROOT}/lease_delete.json"
python3 - "${TMP_ROOT}/lease_delete.json" <<'PY'
import json,sys
obj=json.load(open(sys.argv[1]))
if obj.get("ok") is not True:
    raise SystemExit("lease delete did not return ok=true")
PY
pass_case "daemon-lease-api-delete"

echo "V2 END-TO-END TESTS PASSED"
