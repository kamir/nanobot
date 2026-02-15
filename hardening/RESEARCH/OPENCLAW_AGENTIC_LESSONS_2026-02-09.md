# OpenClaw Agentic Runtime Lessons (PaxMachina) — 2026-02-09

## Scope
This captures what we changed and learned while stabilizing a multi-agent OpenClaw setup (Pax -> workers -> Pax -> channels) across content, finance, SEO, and ops workflows.

## What We Actually Changed

### 1) Stabilized delegation contract and callback flow
- Standardized worker callback contract usage (`CORRELATION_ID`, `STATUS`, `SUMMARY`) and enforced correlation checks.
- Added/used delegation receipts + task ledger tracking (`memory/delegation_receipts.jsonl`, `memory/TASKS.md`) as source of truth.
- Diagnosed case where tasks were DONE but user saw no Telegram response.

### 2) Fixed callback delivery bottleneck
- Root issue: worker completed, but Telegram callback send failed in `scripts/delegate.sh`.
- Practical failure mode: callback send timeout too short for real CLI/gateway latency.
- Fix applied:
  - `DELEGATION_NOTIFY_TIMEOUT` made configurable (default now higher).
  - Added explicit callback failure diagnostics (exit code + truncated CLI output) into `logs/delegate_callbacks.log`.
- Result: callback delivery path confirmed working again.

### 3) Hardened content pipeline process model
- Confirmed workflow target: `inbox -> work -> approved -> published`.
- Enforced one-flow-per-day control to cap token spend and increase predictability.
- Added controls to prevent fallback placeholders from silently becoming "final" output.
- Added/kept strict human-facing contract around publish moves.

### 4) Publish/pipeline behavior improvements
- Added natural-language publish helper path so human instructions map to deterministic actions.
- Added publish guardrail logic: approved->published should include URL metadata handling and stronger checks.

### 5) Repo/path hygiene
- Found duplicate clone/path confusion (`~/workspace/content-pipeline` vs `~/workspace/repos/content-pipeline`).
- Identified this as a major source of “it worked but not here” incidents.
- Standardized automation around one canonical repo path.

### 6) GSC/GA signal integration work
- Validated GSC query/report flows.
- Investigated GA4 service account/property issues.
- Reduced reporting noise; moved toward concise operator-grade weekly output.
- Added property mapping discipline to avoid property-scope/query mismatches.

### 7) Trigger/heartbeat routing policy
- Continued migration from many ad-hoc scripts toward `cron -> trigger -> agent`.
- Reinforced policy: channel messaging should route through Pax (single orchestration boundary).

## Main Pain Points Observed

1. Silent post-success failures
- Worker success != end-user success. Callback/channel send can fail after core task succeeds.

2. State split across files + channels
- If TASKS, receipts, logs, and chat are not tied by one correlation id, incident triage is slow.

3. Runtime drift
- Different runtimes/gateways/CLI latencies cause false “stuck” perception.

4. Repo/path drift
- Multiple clones or outdated paths create hidden split-brain behavior.

5. Fallback semantics
- Auto-fallback must never overwrite valuable content.

6. Prompt intent mapping errors
- Natural language requests can route to wrong tool (example class: stats query interpreted as submit-index workflow).

## What Works Well (Keep)

- Correlation-id-first design.
- Task ledger + receipt journal.
- Stage quotas and daily flow caps.
- Agent-to-agent handoffs only for explicit allowed edges.
- Human-in-the-loop publish decisions with explicit URL capture.

## Enterprise Requirements for a Go Rebuild (Scalbotics)

### Control Plane
- Durable state machine per task:
  - states: queued, running, waiting_callback, callback_sent, callback_failed, done, blocked, error.
- Strong idempotency keys for every external action.
- Explicit retries with exponential backoff + dead letter queue.

### Messaging Plane
- Separate worker execution from user-notification delivery.
- Never block task completion on channel send.
- Keep callback as independent job with retry budget and audit trail.

### Observability
- One canonical event schema:
  - `task_id`, `corr_id`, `from`, `to`, `stage`, `status`, `attempt`, `latency_ms`, `error_code`.
- Structured logs + metrics + trace ids.
- SLOs:
  - delegation success rate
  - callback delivery success rate
  - median/95p end-to-end completion

### Safety/Policy
- Contract validators for every worker response.
- Guardrails preventing fallback placeholders from replacing finalized content.
- Policy engine for allowed handoffs and sensitive actions.

### Content Ops Domain Rules
- One canonical repo root per environment.
- Stage transitions as transactions with validation hooks.
- Publish action requires URL + metadata update or explicit override reason.

### Cost Control
- Native quotas (daily per flow, per agent, per model).
- Prefer local tool execution before LLM.
- Token accounting per task and per pipeline.

## Concrete Implementation Pattern (Go)

1. Ingest trigger command/event.
2. Create task row + correlation id.
3. Execute worker step.
4. Validate worker contract.
5. Persist result payload.
6. Emit callback job.
7. Callback worker sends channel message with retry and idempotency.
8. Mark callback outcome independently.
9. Expose final composite status in API/UI.

## Suggested Next Steps for Scalbotics

1. Build a minimal “delegation + callback” service first (no full orchestration yet).
2. Add a deterministic test harness for:
   - timeout
   - worker done + callback fail
   - callback retry recovery
   - duplicate triggers
3. Implement a canonical path registry to eliminate repo split-brain.
4. Add strict publish transition validator package for content pipelines.
5. Add intent-router tests for common natural language commands.

## Short Learning Summary
The core failure class was not model quality; it was orchestration reliability. The winning pattern is: strict contracts, durable state transitions, decoupled callback delivery, and path/config determinism.
