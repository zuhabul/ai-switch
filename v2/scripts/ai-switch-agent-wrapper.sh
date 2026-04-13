#!/usr/bin/env bash
set -euo pipefail

API_BASE="${AI_SWITCHD_URL:-http://127.0.0.1:4417}"
FRONTEND="${AI_SWITCH_FRONTEND:-}"
TASK_CLASS="${AI_SWITCH_TASK_CLASS:-coding}"
REQUIRED_PROTOCOL="${AI_SWITCH_REQUIRED_PROTOCOL:-}"
PREFERRED_PROVIDERS="${AI_SWITCH_PREFERRED_PROVIDERS:-}"
REQUIRE_TAGS="${AI_SWITCH_REQUIRE_TAGS:-}"
OWNER="${AI_SWITCH_OWNER:-wrapper-$(id -un 2>/dev/null || echo user)}"
REAL_CMD="${AI_SWITCH_REAL_CMD:-}"
PLATFORM_USE="${AI_SWITCH_PLATFORM_USE:-}"
LEASE_TTL_MIN="${AI_SWITCH_LEASE_TTL_MIN:-15}"

if [[ -z "${FRONTEND}" ]]; then
  echo "AI_SWITCH_FRONTEND is required" >&2
  exit 2
fi
if [[ -z "${REAL_CMD}" ]]; then
  REAL_CMD="${FRONTEND}"
fi

if [[ "${1:-}" == "--version" || "${1:-}" == "version" || "${1:-}" == "-V" || "${1:-}" == "--help" || "${1:-}" == "help" ]]; then
  exec "${REAL_CMD}" "$@"
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 2
fi

route_payload="$(python3 - <<PY
import json
print(json.dumps({
  "frontend": "${FRONTEND}",
  "task_class": "${TASK_CLASS}",
  "required_protocol": "${REQUIRED_PROTOCOL}",
  "preferred_providers": [x for x in "${PREFERRED_PROVIDERS}".split(",") if x],
  "require_tags": [x for x in "${REQUIRE_TAGS}".split(",") if x],
  "owner": "${OWNER}",
}))
PY
)"

route_json="$(curl -fsS -X POST "${API_BASE}/v2/route" -H 'content-type: application/json' -d "${route_payload}")"
profile_id="$(python3 - <<PY
import json
obj=json.loads('''${route_json}''')
print(obj.get("profile_id", ""))
PY
)"
if [[ -z "${profile_id}" ]]; then
  echo "route did not return profile_id" >&2
  echo "response: ${route_json}" >&2
  exit 3
fi

lease_payload="$(python3 - <<PY
import json
print(json.dumps({
  "profile_id": "${profile_id}",
  "frontend": "${FRONTEND}",
  "owner": "${OWNER}",
  "ttl_min": int("${LEASE_TTL_MIN}"),
}))
PY
)"
lease_json="$(curl -fsS -X POST "${API_BASE}/v2/leases" -H 'content-type: application/json' -d "${lease_payload}")"
lease_id="$(python3 - <<PY
import json
obj=json.loads('''${lease_json}''')
print(obj.get("id", ""))
PY
)"
if [[ -z "${lease_id}" ]]; then
  echo "lease acquire failed: ${lease_json}" >&2
  exit 4
fi

cleanup() {
  curl -fsS -X DELETE "${API_BASE}/v2/leases?lease_id=${lease_id}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

# For codex/opencode we can apply account directly from existing ai-switch v1 store.
if [[ -n "${PLATFORM_USE}" ]] && command -v ai >/dev/null 2>&1; then
  ai --platform "${PLATFORM_USE}" use "${profile_id}" >/dev/null 2>&1 || ai use "${profile_id}" >/dev/null 2>&1 || true
fi

export AI_SWITCH_PROFILE_ID="${profile_id}"
export AI_SWITCH_LEASE_ID="${lease_id}"

"${REAL_CMD}" "$@"
