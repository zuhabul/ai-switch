# ai-switch v2 Server Deployment

## Installed Components

- Binary: `/home/echo/.local/bin/aiswitch`
- Daemon: `/home/echo/.local/bin/aiswitchd`
- systemd unit: `/etc/systemd/system/aiswitchd.service`
- Traefik route: `/etc/traefik/dynamic/27-workspace-aiswitchd.yml`
- Public endpoint prefix: `https://workspace.zuhabul.com/aiswitch`
- Web console: `https://workspace.zuhabul.com/aiswitch/`

## Health Checks

```bash
curl -fsS http://127.0.0.1:4417/healthz
curl -k -fsS https://workspace.zuhabul.com/aiswitch/healthz
```

## API Examples

```bash
curl -k -fsS https://workspace.zuhabul.com/aiswitch/v2/profiles
curl -k -fsS https://workspace.zuhabul.com/aiswitch/v2/dashboard/summary
curl -k -fsS https://workspace.zuhabul.com/aiswitch/v2/adapters/contract
curl -k -fsS https://workspace.zuhabul.com/aiswitch/metrics
curl -k -fsS -X POST https://workspace.zuhabul.com/aiswitch/v2/route \
  -H 'content-type: application/json' \
  -d '{"frontend":"codex","task_class":"coding","required_protocol":"app_server"}'
curl -k -fsS -X POST https://workspace.zuhabul.com/aiswitch/v2/route/candidates \
  -H 'content-type: application/json' \
  -d '{"frontend":"codex","task_class":"coding"}'
curl -k -fsS -X POST https://workspace.zuhabul.com/aiswitch/v2/incidents \
  -H 'content-type: application/json' \
  -d '{"profile_id":"codex-main","kind":"rate_limit","message":"429","owner":"ops","cooldown_seconds":900}'
```

## Optional API Authentication

Configure one or both:

- `AISWITCHD_API_TOKEN=...` (Bearer auth)
- `AISWITCHD_HMAC_KEYS=ops:secret1,bot:secret2` (HMAC headers)

When enabled, `/v2/*` and `/metrics` require:

- `Authorization: Bearer <token>` OR
- `X-AISWITCH-Key-ID`, `X-AISWITCH-Timestamp`, `X-AISWITCH-Signature`

## Wrapper Integration

Wrappers installed under `~/.local/bin`:

- `multica-codex-aiswitch`
- `multica-claude-aiswitch`
- `multica-opencode-aiswitch`
- `multica-hermes-aiswitch`
- `multica-gemini-aiswitch`
- `multica-openclaw-aiswitch` (requires `openclaw` binary present)
- `multica-copilot-aiswitch`
- `multica-qwen-aiswitch`
- `multica-kimi-aiswitch`
- `multica-aider-aiswitch`

Generic wrapper:

- `ai-switch-agent-wrapper`

## Multica Wiring

`/etc/systemd/system/multica-daemon.service` uses wrapper paths via:

- `MULTICA_CODEX_PATH`
- `MULTICA_CLAUDE_PATH`
- `MULTICA_OPENCODE_PATH`
- `MULTICA_HERMES_PATH`
- `MULTICA_GEMINI_PATH`
- `MULTICA_OPENCLAW_PATH`

The wrappers call:

1. `POST /v2/route`
2. `POST /v2/leases`
3. Execute the real CLI
4. `DELETE /v2/leases?lease_id=...`
