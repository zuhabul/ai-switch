# ai-switch v2 World-Class Migration Task List

## Objective
Build a provider-agnostic AI runtime/auth control plane that supports multi-account routing and professional failover across any agent frontend, any coding CLI, and compatible API providers.

## Scope Definition
- Keep `v1` shell workflow available during migration.
- Build `v2` as modular services with strict profile isolation.
- Support native CLIs and compatibility API protocols under one policy/routing engine.
- Deliver production controls: observability, security, policy, SLOs, and disaster recovery.

## Workstreams

### 1. Architecture and Foundation
- [x] Define canonical domain model: `provider`, `frontend`, `auth_method`, `protocol`, `profile`, `lease`, `policy`.
- [x] Create `v2` module skeleton with typed internal packages.
- [x] Add command binaries: `aiswitch` and `aiswitchd`.
- [ ] Create RFC-0001 with non-goals and failover guarantees.
- [x] Create versioned adapter SDK contract.

### 2. Security and Identity
- [ ] Add secret vault abstraction with keyring + encrypted file fallback.
- [ ] Add secret handles so runtime never logs raw credentials.
- [ ] Add auth sessions lifecycle tracking (created, rotated, revoked).
- [ ] Add account lockout and anomaly detection rules.
- [x] Add per-profile RBAC owner groups.

### 3. Provider and CLI Adapters
- [x] Create built-in capability registry for core providers and frontends.
- [ ] Implement adapter execution contracts: `detect`, `validate`, `refresh`, `launch`, `checkpoint`, `resume`.
- [x] Implement native adapters for Codex, Claude Code, Gemini CLI, Qwen Code, Kimi CLI.
- [x] Implement compatibility adapters for OpenAI-compatible, Anthropic-compatible, Gemini-compatible protocols.
- [x] Implement provider adapters for MiniMax, Z.AI/GLM, xAI, Moonshot API endpoints.

### 4. Routing, Quotas, and Failover
- [x] Implement scoring router with policy, health, cooldown filters.
- [x] Implement cooldown management API.
- [x] Implement profile lease locking for session ownership.
- [ ] Add adaptive rate-limit model from live telemetry.
- [ ] Add circuit breakers per provider/profile/model.
- [ ] Add fallback trees per task class.
- [ ] Add warm standby profile pools.
- [ ] Add budget-aware re-routing and hard stop rules.

### 5. Session and Runtime Orchestration
- [ ] Add process supervision and PTY management.
- [ ] Add checkpoint/resume APIs for managed runtime mode.
- [ ] Add native CLI restart handoff at turn boundaries.
- [ ] Add app-server broker for long-running runtimes.
- [ ] Add run journaling for deterministic replay/debug.

### 6. API, Control Plane, and Integrations
- [x] Add HTTP control API endpoints for profiles, policies, health, route, leases.
- [x] Add management UX frontend served by `aiswitchd`.
- [x] Add route candidate endpoint for multi-account failover planning.
- [x] Add authentication for control API (bearer token and HMAC signed mode).
- [ ] Add webhooks/events for rate-limit and failover incidents.
- [ ] Add Hermes/OpenCode/OpenClaw integration bridges.
- [ ] Add Multica runtime bridge for centralized profile routing.
- [ ] Add Discord/Telegram/WhatsApp control adapters.
- [ ] Add mobile operator dashboard APIs.

### 7. Observability and Reliability
- [ ] Add structured event taxonomy and reason codes.
- [ ] Add Prometheus metrics endpoints.
- [ ] Add OpenTelemetry traces.
- [ ] Add SLO dashboards for success rate, latency, failover time.
- [ ] Add chaos tests for limit events and provider outages.

### 8. Quality and Verification
- [x] Add unit tests for routing, policy, store, adapter registry, and lease behavior.
- [x] Add HTTP API tests for dashboard and route candidate flows.
- [ ] Add integration tests with mocked provider endpoints.
- [ ] Add contract tests for adapters.
- [ ] Add load tests for route API and lease contention.
- [ ] Add security scans and secret leak checks.
- [ ] Add migration tests from v1 profile store.

### 9. Release and Migration
- [ ] Create v1 to v2 profile importer.
- [ ] Create rollback and canary deployment plan.
- [ ] Create runbooks and incident playbooks.
- [ ] Tag v2.0.0 release with compatibility notes.
- [ ] Deprecate v1 commands in staged timeline.

## 30 Professional Universal Functionalities (Target)
1. Multi-profile isolation by provider/frontend/auth/protocol.
2. Cross-provider account registry.
3. Lease-based exclusive session ownership.
4. Cooldown and backoff state machine.
5. Policy engine with allow/deny constraints.
6. Budget cap policy and enforcement.
7. Health telemetry ingestion.
8. Multi-factor route scoring.
9. Protocol constraint routing.
10. Task-class-aware provider routing.
11. Required-tag governance.
12. Native CLI failover orchestration.
13. Managed runtime failover without session loss.
14. Checkpoint and resume orchestration.
15. App-server lifecycle broker.
16. Secret vault abstraction.
17. Credential lifecycle audit trail.
18. Event webhooks for incident automation.
19. Control API for external orchestrators.
20. Adapter capability introspection.
21. Circuit breakers by profile/provider.
22. Warm standby pools.
23. Rate-limit predictive routing.
24. Structured logs and reason codes.
25. Metrics and tracing instrumentation.
26. Chaos failover testing harness.
27. RBAC and owner scoping.
28. Compliance-ready audit exports.
29. Chatops integrations (Discord/Telegram/WhatsApp).
30. Mobile operator support APIs.

## Production Gates
- Gate A: security hardening completed.
- Gate B: adapter contract coverage >= 95%.
- Gate C: failover test pass rate >= 99% in chaos harness.
- Gate D: p95 route decision latency < 20ms on single node.
- Gate E: no critical vulnerabilities in CI scans.

## Execution Sequence
1. Foundation and model freeze.
2. Security and secret vault.
3. Native adapters for top 3 CLIs.
4. Compatibility adapters for API-first providers.
5. Session broker and managed runtime.
6. Integrations and chatops.
7. SLO hardening and canary rollout.
