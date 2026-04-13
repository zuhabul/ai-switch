# Codex Multi-Account Quick Guide

## Daily Commands

```bash
ai list
ai pool-status
ai status
ai rotate
ai ls
ai doctor
```

- `ai list`: all saved Codex accounts
- `ai pool-status`: pool order + human-readable cooldown timers
- `ai status`: detailed 5h + weekly usage/reset breakdown per account
- `ai rotate`: reconcile pool + refresh limits + switch to the healthiest ready account
- `ai ls`: currently active Codex/OpenCode account
- `ai doctor`: environment readiness check

## Add Another Account

```bash
codex logout
codex login --device-auth
ai --platform codex save <account-name>
ai pool-sync
ai pool-status
ai doctor
```

Use a short stable saved name such as:

```bash
ai --platform codex save saabrinamim
ai --platform codex save propertystudiogpt
```

## Start Codex

Use normal Codex startup:

```bash
codex
```

For fully open permissions:

```bash
codex --dangerously-bypass-approvals-and-sandbox
```

`codex --yolo` is not a valid Codex CLI flag.

## Manual Switch

```bash
ai --platform codex use saabrinamim
ai --platform codex use propertystudiogpt
```

## Quarantine A Rate-Limited Account

If you know an account is currently rate-limited:

```bash
ai pool-cooldown propertystudiogpt 86400
ai pool-status
```

That keeps the account in the pool but makes auto-rotation skip it.

## Re-enable The Account

```bash
ai pool-reset-cooldowns propertystudiogpt
ai pool-status
```

## How Auto-Rotation Works

When `codex` is started through the installed wrapper:

1. it picks the next available account from the pool
2. if Codex exits with a rate-limit message, that account is cooled down
3. the wrapper switches to the next available account
4. it resumes the same session automatically

## Recommended Checks

After any account or pool change:

```bash
ai pool-status
ai doctor
```

For a local rotation/resume self-test:

```bash
ai test-rotation
```

For live limit refresh (when no Codex process is active):

```bash
ai status --refresh
```

`ai status --refresh` uses app-server rate-limit RPC and does not switch your active auth file.

If fallback probing is needed while Codex is running:

```bash
ai status --refresh --unsafe
```
