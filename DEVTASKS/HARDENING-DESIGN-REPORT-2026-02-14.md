# GoMikroBot Hardening Design Report

Status: Proposal
Date: 2026-02-14
Source: `hardening/RESEARCH/ARCHITECTURE.md`, `OPENCLAW_AGENTIC_LESSONS_2026-02-09.md`, `OPENCLAW_AGENT_STACK_FINDINGS_2026-02-13.md`, `hardening/SECURITY-RISKS.md`, `hardening/HARDENING-GUIDE.md`

---

## 1. Executive Summary

Three research artifacts describe enterprise bot architecture (ScalBotics), production multi-agent operational failures (OpenClaw/PaxMachina), and legacy security risks (Nanobot Python). This report extracts the insights relevant to GoMikroBot's current state, identifies 9 architectural gaps, and proposes a 4-phase implementation plan.

**Core finding from OpenClaw (line 136):**
> "The core failure class was not model quality; it was orchestration reliability."

GoMikroBot has solid foundations (trace IDs, pub-sub bus, native WhatsApp, deny-list safety, path containment) but lacks the orchestration durability and policy enforcement needed for reliable production operation.

---

## 2. Current State Assessment

### What GoMikroBot already does well

| Capability | Implementation | Source |
|------------|---------------|--------|
| End-to-end trace/correlation IDs | `bus.InboundMessage.TraceID` propagated through timeline | `bus/bus.go`, `timeline/service.go` |
| Shell command safety | Deny-pattern blocklist + allow-list strict mode + timeout | `tools/shell.go` |
| Filesystem path containment | Writes restricted to work repo, `../` blocked | `tools/filesystem.go` |
| Async outbound dispatch | Pub-sub decouples channels from agent loop | `bus/bus.go` |
| Native WhatsApp (no bridge) | `whatsmeow` library, eliminates unauthenticated WebSocket risk | `channels/whatsapp.go` |
| Localhost-only gateway | Default config binds to 127.0.0.1 | `config/config.go` |
| Channel allowlists | WhatsApp allow/deny lists in settings | Timeline settings |
| Web UI with repo management | Full GitHub panel, timeline, chat | `web/timeline.html` |

### What is missing — 9 gaps

| # | Gap | Severity | Research Source |
|---|-----|----------|----------------|
| G1 | No durable task state machine | Critical | OpenClaw Lessons: "Worker success != end-user success" |
| G2 | No decoupled callback delivery | Critical | OpenClaw Findings: callback send failed after task success |
| G3 | No tool risk tiers or approval gates | High | ScalBotics Architecture: Tier 0/1/2 model |
| G4 | No policy engine beyond static deny-list | High | ScalBotics: RBAC+ABAC, delegation scope limits |
| G5 | No cost/token quotas | High | OpenClaw Lessons: "Native quotas per flow, per agent" |
| G6 | No idempotency / message dedup | Medium | ScalBotics: "Idempotent task envelopes mandatory" |
| G7 | Memory/RAG infra exists but disconnected | Medium | ScalBotics: Memory pipeline with embed+index |
| G8 | No structured memory write path | Medium | ScalBotics: "Agents write JSON, humans write Markdown" |
| G9 | No scheduler with concurrency control | Low (for now) | OpenClaw Findings: overlapping heartbeat runs collapsed throughput |

---

## 3. Key Insights from Research

### 3.1 From OpenClaw Agentic Lessons (2026-02-09)

**Insight 1: Separate task execution from notification delivery.**
The most frequent production failure was: worker completed successfully, but the user saw nothing because the channel send (Telegram/WhatsApp) failed. These are two independent jobs with independent retry budgets.

**Insight 2: Correlation-ID-first design.**
Every message, task, callback, and log entry must share a correlation ID. Without it, incident triage across files/channels/logs is impossibly slow.

**Insight 3: One canonical repo path per environment.**
Multiple clones or outdated paths create "split-brain" — work succeeds in one location but isn't visible in another. GoMikroBot already has this partially (settings-based work repo) but has multiple write paths.

**Insight 4: Fallback semantics must be safe.**
Auto-fallback (e.g., placeholder content replacing real content) must never overwrite valuable data. This applies to GoMikroBot's tool execution — failed tool calls should never silently produce placeholder results that look like success.

**Insight 5: Contract validators for every worker response.**
After each tool or LLM call, validate the response against an expected contract before proceeding. Don't trust raw output blindly.

### 3.2 From OpenClaw Agent Stack Findings (2026-02-13)

**Insight 6: Global run lock prevents throughput collapse.**
Overlapping scheduled runs (heartbeat, cron) caused lock contention and cascading timeouts. Solution: `flock`-style global lock, skip on contention.

**Insight 7: Per-worker concurrency caps.**
A single saturated worker (e.g., content generation) blocked all other workflows sharing that path. Workers need independent concurrency limits.

**Insight 8: Heartbeat-only execution policy.**
Agent execution should require an explicit scheduling context. Direct ad-hoc calls bypass safety controls and create untraceable work.

### 3.3 From ScalBotics Architecture (Draft v0.1)

**Insight 9: Tool risk tiers are essential for enterprise safety.**
- Tier 0: read-only internal tools (always allowed)
- Tier 1: controlled write/internal effects (allowed by policy)
- Tier 2: external or high-impact actions (human approval gate)

**Insight 10: Agent privileges cannot exceed delegated human scope.**
The delegation chain must be explicit: `human → agent → tool`. An agent should never have more tool access than the human who triggered it.

**Insight 11: Memory backend: SQLite + sqlite-vec as v1 default.**
Matches GoMikroBot's existing SQLite timeline. Adding `sqlite-vec` for vector search avoids a Qdrant dependency for local installs while keeping the `VectorStore` interface swappable.

**Insight 12: Immutable append-only audit events.**
Every decision logs: actor, delegated principal, action, target, policy result, and trace ID. GoMikroBot's timeline is close but lacks policy decision logging.

### 3.4 From Security Risk Assessment

**Insight 13: The `authorized` field exists but is never enforced.**
The timeline stores authorization status for events, but no code path checks it before tool execution. This is security theater — audit without enforcement.

**Insight 14: Subagent spawning needs oversight.**
Recursive task spawning can bypass rate limits and user oversight. Any future multi-agent work needs explicit delegation receipts and depth limits.

---

## 4. Design Suggestions

### DS-1: Task Envelope with Durable State Machine

Add a `tasks` table to the timeline SQLite database:

```sql
CREATE TABLE tasks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id     TEXT UNIQUE NOT NULL,
    corr_id     TEXT NOT NULL,
    channel     TEXT NOT NULL,
    sender      TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'queued',
    payload     TEXT,
    result      TEXT,
    error       TEXT,
    attempt     INTEGER DEFAULT 0,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

Valid states: `queued → running → callback_pending → done | error`

Every inbound message creates a task row before any processing begins. The agent loop transitions states atomically. If the process crashes, unfinished tasks can be recovered on restart.

### DS-2: Decoupled Callback Delivery

Split the current agent loop into two stages:

1. **Agent stage**: Consume message → LLM call → tool execution → write result to task row → set status `callback_pending`
2. **Callback stage**: Independent goroutine reads `callback_pending` tasks → delivers to channel → retries on failure → sets `done` or `error`

This ensures "worker done but WhatsApp send failed" is recoverable.

### DS-3: Tool Risk Tiers

Extend the `Tool` interface with a `Tier() int` method:

| Tier | Tools | Policy |
|------|-------|--------|
| 0 | `read_file`, `list_dir`, `web_search`, `web_fetch` | Always allowed |
| 1 | `write_file`, `edit_file`, `shell` (allowlisted commands) | Allowed by default, policy can restrict |
| 2 | `shell` (non-allowlisted), any future external API tools | Requires human approval via WhatsApp confirmation or web UI |

For Tier 2, the agent loop pauses, sends an approval request to the channel, and waits for confirmation before executing.

### DS-4: Policy Evaluation Before Tool Execution

Add a `PolicyEngine` interface:

```go
type PolicyEngine interface {
    Evaluate(ctx PolicyContext) PolicyDecision
}

type PolicyContext struct {
    Sender    string
    Channel   string
    Tool      string
    Tier      int
    Arguments map[string]any
}

type PolicyDecision struct {
    Allow  bool
    Reason string
}
```

The v1 implementation evaluates: sender allowlist, tool tier, and `authorized` status from the timeline. The interface allows future OPA/Cedar integration.

### DS-5: Token Accounting and Cost Quotas

Persist token usage per task:

```sql
ALTER TABLE tasks ADD COLUMN prompt_tokens INTEGER DEFAULT 0;
ALTER TABLE tasks ADD COLUMN completion_tokens INTEGER DEFAULT 0;
ALTER TABLE tasks ADD COLUMN total_tokens INTEGER DEFAULT 0;
```

Add a settings-based quota system:
- `daily_token_limit`: max tokens per 24h period (default: 100,000)
- `per_message_token_limit`: max tokens per single message processing (default: 10,000)

Check quota before each LLM call. Return a user-friendly "quota exceeded" message when limits are hit.

### DS-6: Idempotency Keys

Add `idempotency_key` to `InboundMessage`:

```go
type InboundMessage struct {
    // ... existing fields
    IdempotencyKey string
}
```

WhatsApp uses `MessageID`, web UI generates a UUID. Before processing, check `tasks` table for existing `task_id` matching the idempotency key. If found, return the cached result.

### DS-7: Connect Memory/RAG to Agent Loop

The Qdrant/vector infrastructure exists but is disconnected. Wire it in:

1. On each inbound message, embed the query using the provider's embedding endpoint.
2. Search the vector store for top-k (default 5) relevant memory chunks.
3. Inject retrieved chunks into the system prompt, between soul files and conversation history.
4. After agent response, optionally embed and store the exchange as a new memory chunk.

For v1, use `sqlite-vec` instead of Qdrant for zero-dependency local operation. Keep the `VectorStore` interface for future backend swap.

### DS-8: Structured Memory Write Path

Add a `memory` tool to the tool registry:

```go
type MemoryWriteTool struct{}

func (t *MemoryWriteTool) Name() string { return "remember" }
func (t *MemoryWriteTool) Tier() int    { return 1 }
```

The tool writes a JSON record to `MEMORY/` in the workspace:
```json
{
    "schema_version": 1,
    "created_at": "2026-02-14T10:30:00Z",
    "actor": "agent",
    "type": "fact",
    "content": "User prefers concise responses",
    "tags": ["preference", "style"],
    "source_trace_id": "abc-123"
}
```

Records are validated, embedded, and indexed into the vector store.

### DS-9: Scheduler with Concurrency Control (Future)

Add a scheduler package with:
- Cron-triggered task dispatch
- Global run lock (only one scheduled run at a time)
- Per-worker concurrency caps
- Heartbeat context propagation (scheduled vs. ad-hoc execution)

This is deferred until multi-agent or recurring workflow support is needed.

---

## 5. Implementation Plan

### Phase 1: Orchestration Foundation

**Goal:** Make the system crash-recoverable and notification-reliable.

| Task | What | Files | Depends On |
|------|------|-------|------------|
| H-001 | Add `tasks` table to timeline schema | `timeline/schema.go` | — |
| H-002 | Add task CRUD methods to timeline service | `timeline/service.go` | H-001 |
| H-003 | Add `IdempotencyKey` to `InboundMessage` | `bus/bus.go` | — |
| H-004 | Refactor agent loop: create task row on receive, update status on completion | `agent/loop.go` | H-002, H-003 |
| H-005 | Add dedup check in agent loop (skip if task exists for same idempotency key) | `agent/loop.go` | H-004 |
| H-006 | Add callback worker goroutine: reads `callback_pending` tasks, delivers to channel with retry | `agent/loop.go`, `bus/bus.go` | H-004 |
| H-007 | Add task status endpoint to web API | `gateway.go` | H-002 |

**Acceptance criteria:**
- [ ] Every inbound message creates a task row with trace ID
- [ ] Task status progresses: `queued → running → callback_pending → done`
- [ ] Duplicate messages (same idempotency key) return cached result
- [ ] If WhatsApp send fails, task stays `callback_pending` and retries
- [ ] Task status visible in web UI

### Phase 2: Safety and Policy

**Goal:** Enforce authorization and limit blast radius.

| Task | What | Files | Depends On |
|------|------|-------|------------|
| H-008 | Add `Tier() int` method to Tool interface | `tools/tool.go` | — |
| H-009 | Classify existing tools by tier (0/1/2) | `tools/filesystem.go`, `tools/shell.go`, `tools/web.go` | H-008 |
| H-010 | Add `PolicyEngine` interface and v1 implementation | `internal/policy/engine.go` (new) | — |
| H-011 | Wire policy check into agent loop before tool execution | `agent/loop.go` | H-009, H-010 |
| H-012 | Add Tier 2 approval gate: pause, ask user via channel, wait for confirmation | `agent/loop.go`, `bus/bus.go` | H-011 |
| H-013 | Add token usage persistence to tasks table | `timeline/schema.go`, `timeline/service.go` | H-002 |
| H-014 | Add quota check before LLM call | `agent/loop.go` | H-013 |
| H-015 | Add policy decision logging to timeline | `timeline/service.go` | H-010 |

**Acceptance criteria:**
- [ ] Each tool reports its risk tier
- [ ] Tier 2 tools prompt the user for approval before execution
- [ ] Unauthorized senders cannot trigger tool execution
- [ ] Token usage is recorded per task
- [ ] Daily quota exceeded → user-friendly message, no LLM call
- [ ] Policy decisions are logged with trace ID

### Phase 3: Memory and Intelligence

**Goal:** Give the agent persistent memory and retrieval.

| Task | What | Files | Depends On |
|------|------|-------|------------|
| H-016 | Add `sqlite-vec` extension to timeline DB | `timeline/schema.go` | — |
| H-017 | Implement `VectorStore` interface with sqlite-vec backend | `internal/memory/sqlite_vec.go` (new) | H-016 |
| H-018 | Add embedding call to provider (OpenAI `text-embedding-3-small`) | `provider/openai.go` | — |
| H-019 | Wire RAG into context builder: embed query → search → inject top-k into prompt | `agent/context.go`, `agent/loop.go` | H-017, H-018 |
| H-020 | Add `remember` tool for agent to write structured memory | `tools/memory.go` (new) | H-017 |
| H-021 | Add memory chunk schema with versioning | `internal/memory/schema.go` (new) | — |
| H-022 | Index existing soul files as memory chunks on startup | `agent/context.go` | H-017, H-021 |

**Acceptance criteria:**
- [ ] Agent responses incorporate relevant past context via vector search
- [ ] Agent can persist facts/preferences via `remember` tool
- [ ] Memory chunks have versioned schema with source metadata
- [ ] Soul files are searchable alongside dynamic memory
- [ ] Vector search uses local sqlite-vec (no external dependency)

### Phase 4: Scheduling and Multi-Agent (Future)

**Goal:** Enable recurring workflows and agent-to-agent coordination.

| Task | What | Files | Depends On |
|------|------|-------|------------|
| H-023 | Add scheduler package with cron dispatch | `internal/scheduler/scheduler.go` (new) | Phase 1 |
| H-024 | Add global run lock (flock-style) | `internal/scheduler/lock.go` (new) | H-023 |
| H-025 | Add per-worker concurrency caps | `internal/scheduler/scheduler.go` | H-023 |
| H-026 | Add delegation contract (correlation ID, status, summary) | `bus/bus.go` | Phase 1 |
| H-027 | Add delegation receipt logging | `timeline/service.go` | H-026 |

**Acceptance criteria:**
- [ ] Scheduled tasks run under explicit heartbeat context
- [ ] Overlapping schedule ticks are skipped (global lock)
- [ ] Worker concurrency does not exceed configured cap
- [ ] Agent-to-agent delegation is traceable via receipts

---

## 6. Priority Recommendation

**Start with Phase 1 (H-001 through H-007).** It touches files we already work in daily, doesn't change user-facing behavior, and addresses the #1 production failure class identified by OpenClaw. Every subsequent phase depends on durable task tracking.

**Then Phase 2 (H-008 through H-015).** This is the most important safety improvement and directly addresses the critical risks in `SECURITY-RISKS.md`.

Phase 3 is the highest-leverage intelligence improvement. Phase 4 is deferred until multi-agent or recurring workflows are needed.

---

## 7. Risk Assessment

| Risk | Mitigation |
|------|-----------|
| Task table adds write overhead per message | SQLite WAL mode, batch updates, minimal row size |
| Approval gate blocks agent loop | Timeout with safe default (deny after 60s) |
| sqlite-vec not available on all platforms | Fallback to keyword search; keep VectorStore interface |
| Quota check adds latency before LLM call | Single SELECT query, sub-millisecond |
| Callback retry can send duplicate messages | Include idempotency key in outbound, channels dedup |

---

## 8. Dependency on Existing Tasks

This plan extends the existing migration task list (`DEVTASKS/tasks.md`). Mapping:

| Existing Task | Hardening Extension |
|--------------|-------------------|
| TASK-QMD-001 (SQLite Timeline) | H-001 adds `tasks` table to same schema |
| TASK-QMD-003 (Qdrant Integration) | H-017 replaces with sqlite-vec for v1 |
| TASK-008 (Shell Exec) | H-009 adds tier classification |
| TASK-013 (Agent Loop) | H-004, H-005, H-006 refactor the loop |
| TASK-017 (Cron Service) | H-023 through H-025 are the hardened version |

---

## 9. Change Log

- 2026-02-14: Initial report from hardening research analysis.
