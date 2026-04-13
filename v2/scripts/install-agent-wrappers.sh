#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="${1:-$HOME/.local/bin}"
mkdir -p "${INSTALL_DIR}"

WRAPPER_SRC="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/ai-switch-agent-wrapper.sh"
WRAPPER_BIN="${INSTALL_DIR}/ai-switch-agent-wrapper"
install -m 0755 "${WRAPPER_SRC}" "${WRAPPER_BIN}"

write_wrapper() {
  local file="$1"
  local frontend="$2"
  local protocol="$3"
  local real_cmd="$4"
  local platform_use="$5"

  cat > "${INSTALL_DIR}/${file}" <<SH
#!/usr/bin/env bash
set -euo pipefail
export AI_SWITCH_FRONTEND="${frontend}"
export AI_SWITCH_REQUIRED_PROTOCOL="${protocol}"
export AI_SWITCH_REAL_CMD="${real_cmd}"
export AI_SWITCH_PLATFORM_USE="${platform_use}"
exec "${WRAPPER_BIN}" "\$@"
SH
  chmod +x "${INSTALL_DIR}/${file}"
}

write_wrapper "multica-codex-aiswitch" "codex" "app_server" "/home/echo/.local/bin/codex.ai-switch-real" "codex"
write_wrapper "multica-opencode-aiswitch" "opencode" "native_cli" "/home/echo/.local/bin/opencode.real" "opencode"
write_wrapper "multica-claude-aiswitch" "claude_code" "native_cli" "/home/echo/.local/bin/claude.real" ""
write_wrapper "multica-gemini-aiswitch" "gemini_cli" "native_cli" "/home/echo/.local/bin/gemini" ""
write_wrapper "multica-hermes-aiswitch" "hermes" "hermes" "/home/echo/.local/bin/hermes" ""
write_wrapper "multica-openclaw-aiswitch" "openclaw" "native_cli" "openclaw" ""

echo "Installed wrappers in ${INSTALL_DIR}"
