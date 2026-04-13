# ai-switch v2 Production Readiness Checklist

## Security
- [ ] Secrets are never printed in logs.
- [ ] Secrets at rest are encrypted.
- [ ] Control API is authenticated and authorized.
- [ ] Least-privilege scopes validated for every provider token.
- [ ] Rotation and revocation playbooks tested.

## Reliability
- [ ] All critical paths have retries and bounded timeouts.
- [ ] Circuit breaker behavior verified under outage simulation.
- [ ] Cooldown logic verified for 429 and upstream errors.
- [ ] Lease collision handling tested under concurrency.
- [ ] Crash recovery restores consistent state.

## Observability
- [ ] Structured logs include correlation IDs.
- [ ] Metrics include route success/failure and fallback reasons.
- [ ] Traces include adapter and provider spans.
- [ ] Alerting configured for failover storms and high error rates.

## Performance
- [ ] Route API p95 latency measured under load.
- [ ] Memory and file descriptor limits verified.
- [ ] State persistence throughput tested.

## Integrations
- [ ] Codex adapter validated.
- [ ] Claude Code adapter validated.
- [ ] Gemini CLI adapter validated.
- [ ] Qwen adapter validated.
- [ ] Kimi adapter validated.
- [ ] MiniMax adapter validated.
- [ ] Z.AI adapter validated.
- [ ] xAI adapter validated.

## Operational Excellence
- [ ] Canary deployment with rollback tested.
- [ ] Runbooks are complete and reviewed.
- [ ] On-call escalation matrix documented.
- [ ] Incident timeline export available.

## Compliance
- [ ] Audit trail tamper detection enabled.
- [ ] Data retention policy implemented.
- [ ] Access control change logs retained.
