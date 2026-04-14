#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
V2_DIR="${ROOT_DIR}/v2"
WRAPPER="${V2_DIR}/scripts/ai-switch-agent-wrapper.sh"

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

wait_for_health() {
  local port="$1"
  local extra_header="${2:-}"
  local url="http://127.0.0.1:${port}/healthz"
  for _ in $(seq 1 60); do
    if [[ -n "${extra_header}" ]]; then
      if curl -fsS -H "${extra_header}" "${url}" >/dev/null 2>&1; then
        return 0
      fi
    else
      if curl -fsS "${url}" >/dev/null 2>&1; then
        return 0
      fi
    fi
    sleep 0.1
  done
  return 1
}

start_daemon() {
  local port="$1"
  shift
  "${AISWITCHD_BIN}" --addr "127.0.0.1:${port}" "$@" >"${TMP_ROOT}/aiswitchd-${port}.log" 2>&1 &
  DAEMON_PID=$!
  wait_for_health "${port}" || {
    cat "${TMP_ROOT}/aiswitchd-${port}.log" >&2 || true
    fail_case "daemon did not become healthy on port ${port}"
  }
}

stop_daemon() {
  if [[ -n "${DAEMON_PID:-}" ]]; then
    kill "${DAEMON_PID}" >/dev/null 2>&1 || true
    wait "${DAEMON_PID}" >/dev/null 2>&1 || true
    DAEMON_PID=""
  fi
}

route_request_body() {
  local owner="$1"
  cat <<JSON
{"frontend":"codex","task_class":"coding","required_protocol":"app_server","owner":"${owner}"}
JSON
}

lease_post() {
  local port="$1"
  local owner="$2"
  local ttl="$3"
  local profile="$4"
  local auth_header="${5:-}"
  if [[ -n "${auth_header}" ]]; then
    curl -fsS -H "${auth_header}" -X POST "http://127.0.0.1:${port}/v2/leases" \
      -H 'content-type: application/json' \
      -d "{\"profile_id\":\"${profile}\",\"frontend\":\"codex\",\"owner\":\"${owner}\",\"ttl_min\":${ttl}}"
  else
    curl -fsS -X POST "http://127.0.0.1:${port}/v2/leases" \
      -H 'content-type: application/json' \
      -d "{\"profile_id\":\"${profile}\",\"frontend\":\"codex\",\"owner\":\"${owner}\",\"ttl_min\":${ttl}}"
  fi
}

require_cmd go
require_cmd curl
require_cmd jq
require_cmd python3

TMP_ROOT="$(mktemp -d)"
trap 'stop_daemon; rm -rf "${TMP_ROOT}"' EXIT

export GOCACHE="${GOCACHE:-/tmp/ai-switch-gocache}"
export GOMODCACHE="${GOMODCACHE:-/tmp/ai-switch-gomodcache}"
mkdir -p "${GOCACHE}" "${GOMODCACHE}"

BIN_DIR="${TMP_ROOT}/bin"
mkdir -p "${BIN_DIR}"
AISWITCH_BIN="${BIN_DIR}/aiswitch"
AISWITCHD_BIN="${BIN_DIR}/aiswitchd"

export HOME="${TMP_ROOT}/home"
mkdir -p "${HOME}"

go -C "${V2_DIR}" build -o "${AISWITCH_BIN}" ./cmd/aiswitch
go -C "${V2_DIR}" build -o "${AISWITCHD_BIN}" ./cmd/aiswitchd

"${AISWITCH_BIN}" init >/dev/null

"${AISWITCH_BIN}" profile add --id codex-restricted --provider openai --frontend codex --auth chatgpt --protocol app_server --owner-scopes ops --priority 100 --tags prod --budget 20 >/dev/null
"${AISWITCH_BIN}" profile add --id codex-main --provider openai --frontend codex --auth chatgpt --protocol app_server --owner-scopes multica,ops --priority 80 --tags prod,primary --budget 12 >/dev/null
"${AISWITCH_BIN}" profile add --id codex-backup --provider openai --frontend codex --auth chatgpt --protocol app_server --owner-scopes multica,ops --priority 60 --tags prod,backup --budget 8 >/dev/null
"${AISWITCH_BIN}" profile add --id lease-target --provider openai --frontend codex --auth chatgpt --protocol app_server --priority 20 --tags test --budget 2 >/dev/null

"${AISWITCH_BIN}" health set --id codex-restricted --r5m 90 --rh 900 --latency 90 --error 0.1 >/dev/null
"${AISWITCH_BIN}" health set --id codex-main --r5m 60 --rh 600 --latency 110 --error 0.2 >/dev/null
"${AISWITCH_BIN}" health set --id codex-backup --r5m 55 --rh 500 --latency 140 --error 0.3 >/dev/null
"${AISWITCH_BIN}" health set --id lease-target --r5m 30 --rh 300 --latency 150 --error 0.4 >/dev/null

PORT=4477
start_daemon "${PORT}"

# Owner-scope route enforcement
owner_multica_body="$(route_request_body "multica")"
route_multica_json="$(curl -fsS -X POST "http://127.0.0.1:${PORT}/v2/route/candidates" -H 'content-type: application/json' -d "${owner_multica_body}")"
owner_multica_primary="$(jq -r '.primary.profile_id' <<<"${route_multica_json}")"
[[ "${owner_multica_primary}" == "codex-main" ]] || fail_case "owner scope route expected codex-main for multica, got ${owner_multica_primary}"
pass_case "owner-scope-route-multica"

owner_ops_body="$(route_request_body "ops")"
route_ops_json="$(curl -fsS -X POST "http://127.0.0.1:${PORT}/v2/route/candidates" -H 'content-type: application/json' -d "${owner_ops_body}")"
owner_ops_primary="$(jq -r '.primary.profile_id' <<<"${route_ops_json}")"
[[ "${owner_ops_primary}" == "codex-restricted" ]] || fail_case "owner scope route expected codex-restricted for ops, got ${owner_ops_primary}"
pass_case "owner-scope-route-ops"

# Lease idempotent refresh for same owner
lease_a="$(lease_post "${PORT}" "multica" 1 "lease-target")"
lease_a_id="$(jq -r '.id' <<<"${lease_a}")"
lease_a_exp="$(jq -r '.expires_at' <<<"${lease_a}")"
sleep 1
lease_b="$(lease_post "${PORT}" "multica" 5 "lease-target")"
lease_b_id="$(jq -r '.id' <<<"${lease_b}")"
lease_b_exp="$(jq -r '.expires_at' <<<"${lease_b}")"
[[ "${lease_a_id}" == "${lease_b_id}" ]] || fail_case "same-owner lease refresh should keep lease id"
python3 - "${lease_a_exp}" "${lease_b_exp}" <<'PY'
import datetime,sys
a=datetime.datetime.fromisoformat(sys.argv[1].replace('Z','+00:00'))
b=datetime.datetime.fromisoformat(sys.argv[2].replace('Z','+00:00'))
if not b > a:
    raise SystemExit("lease refresh did not extend expiry")
PY
curl -fsS -X DELETE "http://127.0.0.1:${PORT}/v2/leases?lease_id=${lease_b_id}" >/dev/null
pass_case "lease-refresh-same-owner"

# Lease contention stress: exactly one owner should acquire lease-target.
successes_file="${TMP_ROOT}/lease-successes.log"
: >"${successes_file}"
worker_pids=()
for i in $(seq 1 8); do
  (
    code="$(curl -s -o /tmp/lease-${i}.out -w '%{http_code}' -X POST "http://127.0.0.1:${PORT}/v2/leases" \
      -H 'content-type: application/json' \
      -d "{\"profile_id\":\"lease-target\",\"frontend\":\"codex\",\"owner\":\"multica-${i}\",\"ttl_min\":3}")"
    if [[ "${code}" == "201" ]]; then
      cat "/tmp/lease-${i}.out" | jq -r '.id' >>"${successes_file}"
    fi
  ) &
  worker_pids+=("$!")
done
for pid in "${worker_pids[@]}"; do
  wait "${pid}"
done
success_count="$(grep -c . "${successes_file}" || true)"
[[ "${success_count}" == "1" ]] || fail_case "lease contention expected 1 winner, got ${success_count}"
winner_lease_id="$(head -n1 "${successes_file}")"
curl -fsS -X DELETE "http://127.0.0.1:${PORT}/v2/leases?lease_id=${winner_lease_id}" >/dev/null
pass_case "lease-contention-single-winner"

# Incident-driven failover and health degradation.
before_failover="$(curl -fsS -X POST "http://127.0.0.1:${PORT}/v2/route" -H 'content-type: application/json' -d "${owner_multica_body}" | jq -r '.profile_id')"
[[ "${before_failover}" == "codex-main" ]] || fail_case "expected codex-main before incident, got ${before_failover}"

incident_resp="$(curl -fsS -X POST "http://127.0.0.1:${PORT}/v2/incidents" -H 'content-type: application/json' -d '{"profile_id":"codex-main","kind":"rate_limit","message":"429 during deep test","owner":"multica","cooldown_seconds":180}')"
incident_id="$(jq -r '.id' <<<"${incident_resp}")"
[[ -n "${incident_id}" && "${incident_id}" != "null" ]] || fail_case "incident id missing"

after_failover="$(curl -fsS -X POST "http://127.0.0.1:${PORT}/v2/route" -H 'content-type: application/json' -d "${owner_multica_body}" | jq -r '.profile_id')"
[[ "${after_failover}" == "codex-backup" ]] || fail_case "expected codex-backup after incident cooldown, got ${after_failover}"

health_codex_main="$(curl -fsS "http://127.0.0.1:${PORT}/v2/health" | jq -r '.["codex-main"].remaining_requests_5min')"
[[ "${health_codex_main}" == "0" ]] || fail_case "expected codex-main remaining_requests_5min=0 after rate_limit incident, got ${health_codex_main}"
pass_case "incident-cooldown-failover-health"

# Metrics and contract coverage
contract_json="$(curl -fsS "http://127.0.0.1:${PORT}/v2/adapters/contract")"
methods_count="$(jq -r '.methods | length' <<<"${contract_json}")"
[[ "${methods_count}" -ge 6 ]] || fail_case "adapter contract expected >=6 methods, got ${methods_count}"
metrics_text="$(curl -fsS "http://127.0.0.1:${PORT}/metrics")"
grep -q '^aiswitch_incidents_total ' <<<"${metrics_text}" || fail_case "metrics missing aiswitch_incidents_total"
pass_case "contract-and-metrics"

# Basic route latency SLO check (local p95)
python3 - "${PORT}" <<'PY'
import json, time, urllib.request, statistics, sys
port = int(sys.argv[1])
url = f"http://127.0.0.1:{port}/v2/route"
body = json.dumps({"frontend":"codex","task_class":"coding","required_protocol":"app_server","owner":"multica"}).encode()
durs=[]
for _ in range(80):
    t=time.perf_counter()
    req=urllib.request.Request(url, data=body, headers={"content-type":"application/json"}, method="POST")
    with urllib.request.urlopen(req, timeout=5) as resp:
        resp.read()
    durs.append((time.perf_counter()-t)*1000.0)
durs.sort()
p95 = durs[int(0.95*len(durs))-1]
print(f"p95_ms={p95:.2f}")
if p95 > 60:
    raise SystemExit(f"p95 route latency too high: {p95:.2f}ms")
PY
pass_case "route-latency-p95"

# Restart daemon in secure mode and verify bearer + hmac.
stop_daemon
AUTH_PORT=4478
start_daemon "${AUTH_PORT}" --api-token deep-token --hmac-keys ops:deep-secret

code_noauth="$(curl -s -o /tmp/noauth-secure.out -w '%{http_code}' "http://127.0.0.1:${AUTH_PORT}/v2/dashboard/summary" || true)"
[[ "${code_noauth}" == "401" ]] || fail_case "secure mode expected 401 without auth, got ${code_noauth}"

code_bearer="$(curl -s -o /tmp/bearer-secure.out -w '%{http_code}' -H 'Authorization: Bearer deep-token' "http://127.0.0.1:${AUTH_PORT}/v2/dashboard/summary" || true)"
[[ "${code_bearer}" == "200" ]] || fail_case "secure mode expected 200 with bearer auth, got ${code_bearer}"
pass_case "secure-bearer-auth"

read -r hts hsig < <(python3 - <<'PY'
import hashlib,hmac,time
method='GET'
path='/v2/dashboard/summary'
ts=str(int(time.time()))
body_hash=hashlib.sha256(b'').hexdigest()
payload='\n'.join([method,path,ts,body_hash]).encode()
sig=hmac.new(b'deep-secret',payload,hashlib.sha256).hexdigest()
print(ts, sig)
PY
)
code_hmac_get="$(curl -s -o /tmp/hmac-get.out -w '%{http_code}' \
  -H "X-AISWITCH-Key-ID: ops" \
  -H "X-AISWITCH-Timestamp: ${hts}" \
  -H "X-AISWITCH-Signature: ${hsig}" \
  "http://127.0.0.1:${AUTH_PORT}/v2/dashboard/summary" || true)"
[[ "${code_hmac_get}" == "200" ]] || fail_case "secure mode expected 200 with HMAC GET, got ${code_hmac_get}"
pass_case "secure-hmac-get"

route_body_secure="$(route_request_body "multica")"
read -r pts psig < <(python3 - "${route_body_secure}" <<'PY'
import hashlib,hmac,time,sys
body=sys.argv[1].encode()
method='POST'
path='/v2/route'
ts=str(int(time.time()))
body_hash=hashlib.sha256(body).hexdigest()
payload='\n'.join([method,path,ts,body_hash]).encode()
sig=hmac.new(b'deep-secret',payload,hashlib.sha256).hexdigest()
print(ts, sig)
PY
)
code_hmac_post="$(curl -s -o /tmp/hmac-post.out -w '%{http_code}' -X POST "http://127.0.0.1:${AUTH_PORT}/v2/route" \
  -H 'content-type: application/json' \
  -H "X-AISWITCH-Key-ID: ops" \
  -H "X-AISWITCH-Timestamp: ${pts}" \
  -H "X-AISWITCH-Signature: ${psig}" \
  -d "${route_body_secure}" || true)"
[[ "${code_hmac_post}" == "200" ]] || fail_case "secure mode expected 200 with HMAC POST, got ${code_hmac_post}"
pass_case "secure-hmac-post"

# Wrapper auth/token flow against secure daemon.
helper_cmd="${TMP_ROOT}/print_wrapper_env.sh"
cat > "${helper_cmd}" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
echo "${AI_SWITCH_PROFILE_ID:-}:${AI_SWITCH_LEASE_ID:-}"
SH
chmod +x "${helper_cmd}"

wrapper_out="$(
  AI_SWITCHD_URL="http://127.0.0.1:${AUTH_PORT}" \
  AI_SWITCH_FRONTEND="codex" \
  AI_SWITCH_REQUIRED_PROTOCOL="app_server" \
  AI_SWITCH_OWNER="multica" \
  AI_SWITCH_REAL_CMD="${helper_cmd}" \
  AI_SWITCH_API_TOKEN="deep-token" \
  "${WRAPPER}"
)"
wrapper_profile="$(cut -d: -f1 <<<"${wrapper_out}")"
wrapper_lease="$(cut -d: -f2 <<<"${wrapper_out}")"
[[ -n "${wrapper_profile}" && -n "${wrapper_lease}" ]] || fail_case "wrapper did not expose profile/lease env vars"
[[ "${wrapper_profile}" == "codex-backup" || "${wrapper_profile}" == "codex-main" ]] || fail_case "wrapper picked unexpected profile ${wrapper_profile}"

active_leases_after_wrapper="$(curl -fsS -H 'Authorization: Bearer deep-token' "http://127.0.0.1:${AUTH_PORT}/v2/leases" | jq -r 'length')"
[[ "${active_leases_after_wrapper}" == "0" ]] || fail_case "wrapper cleanup expected 0 active leases, got ${active_leases_after_wrapper}"
pass_case "wrapper-secure-token-flow"

echo "V2 DEEP HARDENING TESTS PASSED"
