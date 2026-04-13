#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROXY="${ROOT_DIR}/seamless/openai_keypool_proxy.py"

TEST_ROOT="$(mktemp -d)"
trap 'rm -rf "${TEST_ROOT}"' EXIT

UPSTREAM_PORT=19091
PROXY_PORT=19092
UPSTREAM_LOG="${TEST_ROOT}/upstream.log"
PROXY_LOG="${TEST_ROOT}/proxy.log"
CONF="${TEST_ROOT}/seamless-proxy.toml"
KEYS="${TEST_ROOT}/keys.txt"

cat > "${KEYS}" <<'EOF'
key-1
key-2
EOF

cat > "${CONF}" <<EOF
listen_host = "127.0.0.1"
listen_port = ${PROXY_PORT}
upstream_base_url = "http://127.0.0.1:${UPSTREAM_PORT}/v1"
client_api_key = "local-proxy-client-key"
retry_statuses = [429, 500, 502, 503, 504]
max_attempts = 2
request_timeout_seconds = 30
keys_file = "${KEYS}"
EOF

python3 - <<'PY' > "${UPSTREAM_LOG}" 2>&1 &
from http.server import BaseHTTPRequestHandler, HTTPServer
import json

class H(BaseHTTPRequestHandler):
    def do_POST(self):
        auth = self.headers.get("Authorization", "")
        body = self.rfile.read(int(self.headers.get("Content-Length", "0")) or 0)
        if auth == "Bearer key-1":
            self.send_response(429)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"error": {"message": "rate limited key-1"}}).encode())
            return
        if auth == "Bearer key-2":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"ok": True, "used": "key-2", "len": len(body)}).encode())
            return
        self.send_response(401)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps({"error": {"message": "unexpected auth"}}).encode())

server = HTTPServer(("127.0.0.1", 19091), H)
server.serve_forever()
PY
UPSTREAM_PID=$!

python3 "${PROXY}" --config "${CONF}" > "${PROXY_LOG}" 2>&1 &
PROXY_PID=$!

cleanup_pids() {
  kill "${PROXY_PID}" "${UPSTREAM_PID}" 2>/dev/null || true
}
trap 'cleanup_pids; rm -rf "${TEST_ROOT}"' EXIT

# wait for proxy health
for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:${PROXY_PORT}/health" >/dev/null 2>&1; then
    break
  fi
  sleep 0.1
done

RESP="$(
  curl -sS "http://127.0.0.1:${PROXY_PORT}/v1/responses" \
    -H "Authorization: Bearer local-proxy-client-key" \
    -H "Content-Type: application/json" \
    -d '{"model":"gpt-4.1-mini","input":"ping"}'
)"

echo "${RESP}" | jq -e '.ok == true and .used == "key-2"' >/dev/null

echo "PASS: proxy retried from key-1 to key-2 on 429"

