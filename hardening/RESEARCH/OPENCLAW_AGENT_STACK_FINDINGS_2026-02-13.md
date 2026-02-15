# OpenClaw Agent Stack Findings (2026-02-13)

## Scope
Operational findings from PaxMachina production runtime on `clawdia@158.220.124.180`, with focus on heartbeat orchestration, delegation reliability, content pipeline throughput, and social handoff behavior.

## Agent Stack (Current Build)

### Core topology
- Coordinator: `paxmachina`
- Specialists:
  - `worker-code` (Forge)
  - `worker-research` (Archon)
  - `worker-seo` (Oracle)
  - `worker-content` (Scribe)
  - `worker-review` (Lecter)
  - `worker-monitor` (Sentinel)

### Runtime model
- OpenClaw gateway service (`openclaw-gateway`) running under systemd user service.
- Agent workspaces split by responsibility (`workspace-*`) to reduce cross-domain context bleed.
- Delegation canonicalized through shell wrappers:
  - `delegate.sh` (primary inter-agent RPC)
  - `agent_task_exec.sh` (file-artifact contract wrapper)
  - `agent_chain_exec.sh` (allowed edge-controlled A2A chain)

### Control-plane principles
- Pax is the orchestrator and policy gate.
- Heartbeat drives recurring work (`content_team_heartbeat.sh` + `HEARTBEAT_TASKS.md`).
- File-based handoff markers are used in content/social flow (`.handoff_scribe`, `.source_sha256`).
- Task receipts and callback logs provide postmortem traceability (`delegation_receipts.jsonl`, callback logs).

## Hardening Changes Applied

### 1) Heartbeat-only delegation enforcement
Implemented guardrails so agent execution must run under heartbeat context unless explicit break-glass is set.

- Enforced in:
  - `workspace/scripts/delegate.sh`
  - `workspace/scripts/agent_task_exec.sh`
  - `workspace/scripts/agent_chain_exec.sh`
- Required runtime context:
  - `HEARTBEAT_CONTEXT=1`
  - `HEARTBEAT_ONLY_HANDOFF=1`
- Break-glass override (manual, explicit):
  - `ALLOW_DIRECT_AGENT_CALLS=1`

### 2) Heartbeat context propagation
`content_team_heartbeat.sh` now executes steps with heartbeat context exported, so managed runs pass policy while ad-hoc direct runs are blocked.

### 3) Overlap prevention (single active heartbeat run)
Added global run lock to `content_team_heartbeat.sh`:
- If previous run is active, next schedule exits immediately with `heartbeat skip reason=lock-held`.
- Lock strategy:
  - `flock` when available
  - `mkdir` fallback when `flock` is unavailable

This enforces: one heartbeat run at a time; overlapping cron ticks are skipped.

## Key Findings

### A) “Heartbeat is running” != “work is done”
Heartbeat schedule was active every 15 minutes, but long-running sub-steps caused lock contention and delayed downstream tasks.

### B) Social handoff correctness vs completion
For published content social flow:
- `.handoff_scribe` matched `.source_sha256` correctly (handoff written).
- `linkedin.md` and `x.md` remained placeholders for affected slug.
- Conclusion: Oracle handoff stage succeeded, Scribe completion failed/timed out.

### C) Throughput collapse caused by concurrent long worker-content jobs
Observed simultaneous long-running `worker-content` delegations from:
- content pipeline rewrite paths
- social post generation paths

Shared worker path caused queueing/lock pressure and repeated timeout patterns.

### D) OpenClaw CLI exit behavior contributes to latency
On remote, direct probes showed commands returning output but not exiting cleanly until outer timeout (`rc=124`) in some paths. This amplifies wall-clock occupancy of wrappers and increases overlap pressure.

## Root-Cause Summary
- Primary bottleneck: saturated `worker-content` execution path under concurrent workloads.
- Secondary bottleneck: prior lack of global heartbeat serialization allowed overlapping scheduler invocations.
- Result: repeated `lock-held`, `rc=124`, and stale handoff markers despite scheduler liveness.

## Operational Guidance

### Mandatory
- Keep heartbeat-only execution policy enabled.
- Keep global heartbeat lock enabled (no concurrent heartbeat runs).
- Treat `ALLOW_DIRECT_AGENT_CALLS=1` as emergency-only and temporary.

### Recommended next controls
- Add per-worker concurrency caps (especially `worker-content`) at orchestration layer.
- Separate heavy rewrite workflows from social generation windows (time-sliced queues).
- Add watchdog metric line per step: start_ts, end_ts, duration_sec, rc, queue_depth.
- Alert when same handoff marker persists beyond one full heartbeat interval.

## Evidence Snapshot (high-level)
- Process tables showed overlapping heartbeat and long-running social/content delegates.
- Heartbeat logs showed frequent `skip reason=interval` plus `fail rc=124` and `lock-held` entries.
- Social logs showed repeated Oracle handoffs with incomplete Scribe finalization on target slug.
- Remote validation confirmed global heartbeat lock now skips overlapping schedule runs.

## Notes
This document is intended as a recurring research artifact for ScalBotics operations and agent-stack evolution.
