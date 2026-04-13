# ai-switch

A small CLI for switching OpenAI accounts between Codex and OpenCode using a
shared local store. It keeps one saved profile per account name and can apply
it to both platforms.

## Requirements

- `bash`
- `jq`

## Install

```bash
git clone <repo-url> ~/ai-switch
ln -sf ~/ai-switch/ai ~/.local/bin/ai
```

Make sure `~/.local/bin` is in your `PATH`.

## Usage

```bash
ai ls
ai list
ai save work
ai use work
ai usage
ai status
ai rotate
ai delete work
ai pool-set work personal backup
ai pool-status
ai codex --search
ai codex-install-wrapper
ai --platform codex save work
ai --platform opencode use work
```

### Notes

- Without `--platform`, `save` and `use` apply to all available platforms.
- Missing platforms are skipped with a warning unless explicitly requested.
- `use` overwrites active auth files without creating backups.
- `delete` removes a saved account from the shared store.
- `usage` shows a compact one-line summary for the active account.
- `status` shows per-account 5h and weekly usage, reset times, cooldown state, source freshness, and account IDs.
- `status --refresh` uses Codex app-server JSON-RPC (`account/rateLimits/read`) per saved account in an isolated temp `CODEX_HOME`, so active sessions are not disturbed.
- `rotate` auto-reconciles the pool with saved accounts, refreshes account limits (default), and switches to the healthiest ready account (lowest weekly/5h usage score).
- `rotate` applies the selected account to all available platforms (Codex and OpenCode) so account state stays aligned.
- `ai codex` is a managed interactive launcher for Codex CLI. It rotates through a configured pool of saved accounts when Codex exits with a rate-limit message, then restarts with `codex resume <session-id>` so the conversation continues.
- Managed Codex requires file-based Codex auth at `~/.codex/auth.json`. If your Codex install is using keyring storage, set `cli_auth_credentials_store = "file"` in your Codex config and sign in again.
- Managed Codex only intercepts interactive `codex` and `codex resume` flows. Other subcommands like `codex login`, `codex logout`, `codex exec`, and `codex review` pass through to the real Codex CLI unchanged.
- True no-restart failover requires request-level retry/failover (proxy mode below), not process-level wrapper rotation.

## Codex Rotation Pool

Configure the saved accounts that `ai codex` is allowed to rotate through:

```bash
ai pool-set work personal backup
ai pool-add spare
ai pool-remove backup
ai pool-reset-cooldowns
ai pool-list
ai pool-status
```

`pool-status` shows the current order, the last selected account, and active cooldowns for accounts that recently hit a limit.

For detailed limits UI, use:

```bash
ai status
ai status --refresh
ai status --refresh --unsafe
ai rotate
ai rotate --refresh
```

`--unsafe` allows fallback tmux probing while Codex is running if app-server refresh fails for any account.

## Managed Codex

Launch Codex through the wrapper instead of calling `codex` directly:

```bash
ai codex
ai codex --search
ai codex resume --last
ai codex-wrapper-status
ai codex-install-wrapper
```

Behavior:

- The wrapper selects the next available account from the pool and writes it to `~/.codex/auth.json`.
- It records the active Codex session ID from `~/.codex/sessions/`.
- If Codex exits with a recognizable rate-limit message, the wrapper marks that account on cooldown, switches to the next available account, and runs `codex resume <session-id>`.
- If you leave a managed Codex run with `Ctrl+C`, the wrapper keeps that session pinned and the next plain `codex` launch in the same working tree resumes the exact chat instead of starting a fresh one.
- Session detection is cwd-aware, so unrelated newer session files elsewhere in `~/.codex/sessions/` cannot steal the resume target.
- If no alternate account is available, the wrapper exits with an error and leaves the cooldown state on disk.

## Truly Seamless Mode (No Process Restart)

If you want failover without Codex process restarts, use the local keypool proxy in `seamless/`.

Why this works:

- Wrapper mode rotates after Codex exits on a limit.
- Proxy mode retries upstream requests across keys before returning failure.
- Codex stays connected to one local endpoint (`http://127.0.0.1:8788/v1`), so 429 failover happens behind the scenes.

Requirements:

- OpenAI API keys with `responses` scope (one key per account/project in your pool).

Setup:

```bash
./seamless/setup_codex_seamless.sh
```

Then:

1. Put one OpenAI key per line in `~/.config/ai-switch/openai-keypool.keys`
2. Set `client_api_key` in `~/.config/ai-switch/seamless-proxy.toml`
3. Export the same client key for Codex:

```bash
export CODEX_SEAMLESS_PROXY_KEY="replace-with-your-client_api_key"
```

Start proxy:

```bash
python3 ./seamless/openai_keypool_proxy.py --config ~/.config/ai-switch/seamless-proxy.toml
```

Run Codex with seamless profile:

```bash
codex -p seamless
```

## Install As Your `codex` Command

If you want the managed launcher to become your normal `codex` command:

```bash
ai codex-wrapper-status
ai codex-install-wrapper
```

This replaces `~/.local/bin/codex` with a tiny wrapper that calls `ai codex "$@"`, and saves the previous entrypoint as:

```bash
~/.local/bin/codex.ai-switch-real
```

To undo it:

```bash
ai codex-uninstall-wrapper
```

Why this survives Codex CLI updates:

- The backup entrypoint still points to the real Codex CLI installation.
- Normal `npm install -g @openai/codex@latest` updates the underlying CLI package, not `~/.local/bin/codex.ai-switch-real`.
- The managed wrapper keeps calling that preserved real entrypoint, so rate-limit rotation keeps working after package updates.

If you change your Node installation layout or rebuild your personal `~/.local/bin/codex` wrapper from scratch, rerun `ai codex-install-wrapper`.

## Multi-Account Setup

One-time setup for multiple ChatGPT accounts:

1. Make sure Codex is using file auth storage.

```toml
# ~/.codex/config.toml
cli_auth_credentials_store = "file"
```

2. Sign in with the first ChatGPT account and save it.

```bash
codex login
ai --platform codex save work
```

3. Sign out, sign in with the next account, and save that too.

```bash
codex logout
codex login
ai --platform codex save personal
```

4. Build the rotation pool.

```bash
ai pool-set work personal
ai pool-status
```

5. Start Codex through the managed launcher.

```bash
ai codex
```

Or install the wrapper once and keep using plain `codex`.

## Testing

Local regression test:

```bash
./tests/managed_codex_rotation_test.sh
./tests/managed_codex_rotation_matrix_test.sh
./tests/seamless_proxy_rotation_test.sh
./tests/status_ui_test.sh
```

That test simulates:

- account 1 starts a Codex session
- Codex exits with a usage-limit message
- the wrapper switches to account 2
- the wrapper resumes the same session ID automatically

## Storage

Default locations:

- Saved accounts: `~/.local/share/ai-switch/credentials/`
- Rotation state: `~/.local/share/ai-switch/state/codex-pool.json`
- Managed Codex logs: `~/.local/share/ai-switch/runtime/codex/`
- Codex auth: `~/.codex/auth.json`
- OpenCode auth: `~/.local/share/opencode/auth.json`

You can override paths with environment variables:

- `AI_SWITCH_HOME` (default `~/.local/share/ai-switch`)
- `CODEX_HOME` (default `~/.codex`)
- `CODEX_AUTH_FILE`
- `OPENCODE_AUTH_FILE`

## Security

This repository is meant to be shared. Do not commit auth files. The `.gitignore`
excludes common secret paths and JSON auth artifacts.

## v2 (Universal Control Plane)

`v1` remains available as the shell-based workflow in `./ai`.

`v2` introduces a typed service layer and daemon for professional multi-account
orchestration across providers, CLIs, and agent frontends.

Current `v2` implementation path:

- `v2/cmd/aiswitch` (operator CLI)
- `v2/cmd/aiswitchd` (HTTP control plane)
- `v2/internal/model` (core domain model)
- `v2/internal/service` (profiles, policies, health, leases, routing)
- `v2/internal/router` (multi-factor route scoring)
- `v2/internal/policy` (provider/auth/tag governance)
- `v2/internal/adapter` (capability registry for providers/frontends)

### Build and Test v2

```bash
cd v2
go test ./...
go build ./cmd/aiswitch ./cmd/aiswitchd
```

### Example v2 workflow

```bash
cd v2
go run ./cmd/aiswitch init
go run ./cmd/aiswitch adapters
go run ./cmd/aiswitch profile add --id codex-main --provider openai --frontend codex --auth chatgpt --protocol app_server --priority 8 --tags prod,primary
go run ./cmd/aiswitch health set --id codex-main --r5m 50 --rh 500 --latency 120 --error 0.2
go run ./cmd/aiswitch route --frontend codex --task coding --protocol app_server
go run ./cmd/aiswitchd --addr 127.0.0.1:4417
```

For migration planning and production gates, see:

- `docs/v2-world-class-migration-tasklist.md`
- `docs/v2-production-readiness-checklist.md`
