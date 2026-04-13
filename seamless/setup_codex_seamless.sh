#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONF_DIR="${HOME}/.config/ai-switch"
CONF_FILE="${CONF_DIR}/seamless-proxy.toml"
KEYS_FILE="${CONF_DIR}/openai-keypool.keys"
CODEX_CONF="${HOME}/.codex/config.toml"

mkdir -p "${CONF_DIR}"

if [[ ! -f "${CONF_FILE}" ]]; then
  cp "${BASE_DIR}/seamless-proxy.example.toml" "${CONF_FILE}"
  chmod 600 "${CONF_FILE}" || true
  echo "Created ${CONF_FILE}"
else
  echo "Using existing ${CONF_FILE}"
fi

if [[ ! -f "${KEYS_FILE}" ]]; then
  cat > "${KEYS_FILE}" <<'EOF'
# Put one OpenAI API key per line (from each account/project).
# Example:
# sk-...
# sk-...
EOF
  chmod 600 "${KEYS_FILE}" || true
  echo "Created ${KEYS_FILE}"
else
  echo "Using existing ${KEYS_FILE}"
fi

mkdir -p "$(dirname "${CODEX_CONF}")"
touch "${CODEX_CONF}"

if ! grep -q '^\[model_providers\.seamlessproxy\]' "${CODEX_CONF}"; then
  cat >> "${CODEX_CONF}" <<'EOF'

# --- ai-switch seamless proxy ---
[model_providers.seamlessproxy]
name = "Seamless OpenAI Keypool Proxy"
base_url = "http://127.0.0.1:8788/v1"
env_key = "CODEX_SEAMLESS_PROXY_KEY"
wire_api = "responses"

[profiles.seamless]
model_provider = "seamlessproxy"
# keep your preferred model here; override with -m if needed
model = "gpt-5-codex"
# --- end ai-switch seamless proxy ---
EOF
  echo "Updated ${CODEX_CONF} with profile 'seamless'."
else
  echo "Codex seamless profile already present in ${CODEX_CONF}"
fi

client_key="$(awk -F'=' '/^client_api_key/ {gsub(/[ "]/, "", $2); print $2}' "${CONF_FILE}" || true)"
if [[ -n "${client_key}" && "${client_key}" != "replace-with-strong-local-key" ]]; then
  echo ""
  echo "Export this in your shell before running codex seamless mode:"
  echo "  export CODEX_SEAMLESS_PROXY_KEY='${client_key}'"
else
  echo ""
  echo "Next steps:"
  echo "1) Set a strong client_api_key in ${CONF_FILE}"
  echo "2) Add your 7 upstream OpenAI API keys to ${KEYS_FILE} (one per line)"
  echo "3) export CODEX_SEAMLESS_PROXY_KEY='<same client_api_key>'"
fi

echo ""
echo "Start proxy:"
echo "  python3 ${BASE_DIR}/openai_keypool_proxy.py --config ${CONF_FILE}"
echo ""
echo "Run Codex via seamless profile:"
echo "  codex -p seamless"

