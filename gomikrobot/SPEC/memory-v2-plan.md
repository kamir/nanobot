# Memory v2 Implementation Plan

> Date: 2026-02-16 | Depends on: memory-architecture-v2.md

---

## Implementation Order

Each step is independently testable. No step exceeds 2 days.

### Step 1: Schema + Working Memory Store
**Files:** `internal/timeline/schema.go`, `internal/memory/working.go`, `internal/memory/working_test.go`

1. Add `working_memory` table to `schema.go` EnsureSchema()
2. Implement `WorkingMemoryStore` with Load/Save methods
3. Write tests: save/load, resource vs thread scoping, fallback behavior, upsert

**Acceptance:** `go test ./internal/memory/... -run TestWorking` passes

### Step 2: Observer + Reflector
**Files:** `internal/memory/observer.go`, `internal/memory/observer_test.go`

1. Define `Observation` struct, `ObserverConfig`, `Observer`
2. Implement `Observe(ctx, sessionID, messages)` — calls LLM, parses observations
3. Implement `Reflect(ctx, sessionID, observations)` — calls LLM, consolidates
4. Implement `ShouldObserve(sessionID)` — checks unobserved message count
5. Implement `LoadObservations(sessionID)` — retrieves ordered observation log
6. Add `observations` table to `schema.go`
7. Write tests with mock LLM provider

**Acceptance:** `go test ./internal/memory/... -run TestObserv` passes

### Step 3: ER1 Client
**Files:** `internal/memory/er1.go`, `internal/memory/er1_test.go`

1. Implement ER1 HTTP client: Authenticate(), FetchMemories(), SyncOnce()
2. Implement SyncLoop(ctx) — periodic sync goroutine
3. Write tests with httptest mock server

**Acceptance:** `go test ./internal/memory/... -run TestER1` passes

### Step 4: Update Lifecycle Policies
**Files:** `internal/memory/lifecycle.go`

1. Add `er1:` and `observation:` to `DefaultPolicies()` (both TTL: 0)
2. Update `Stats()` source grouping CASE statement

**Acceptance:** Existing lifecycle tests still pass

### Step 5: Config + Schema Updates
**Files:** `internal/config/config.go`, `internal/timeline/schema.go`

1. Add `ER1Config` struct to config (URL, APIKey, UserID, SyncInterval)
2. Add `ObserverConfig` to config (Enabled, Model, MessageThreshold, MaxObservations)
3. Add env var mappings: `MIKROBOT_ER1_*`, `MIKROBOT_OBSERVER_*`
4. Ensure `working_memory` and `observations` tables created in EnsureSchema

**Acceptance:** `go build ./cmd/gomikrobot` compiles

### Step 6: Wire into Agent Loop + Context Builder
**Files:** `internal/agent/loop.go`, `internal/agent/context.go`

1. Add `WorkingMemoryStore`, `Observer` to `LoopOptions`
2. In context builder: load working memory for current resource/thread
3. In context builder: load observations for current session
4. After agent response: trigger observer check (non-blocking)
5. Register `update_working_memory` tool if working memory enabled

**Acceptance:** `go build ./cmd/gomikrobot` compiles, gateway starts

### Step 7: Wire into Gateway
**Files:** `cmd/gomikrobot/cmd/gateway.go`

1. Initialize WorkingMemoryStore after MemoryService
2. Initialize ER1Client if configured, start SyncLoop goroutine
3. Initialize Observer if configured
4. Pass all new components to LoopOptions
5. Add shutdown cleanup for ER1 sync and observer

**Acceptance:** Gateway starts cleanly with all components, `go build` succeeds

---

## File Change Matrix

| File | Step | Action | Lines Est. |
|------|------|--------|-----------|
| `internal/memory/working.go` | 1 | NEW | ~120 |
| `internal/memory/working_test.go` | 1 | NEW | ~100 |
| `internal/memory/observer.go` | 2 | NEW | ~250 |
| `internal/memory/observer_test.go` | 2 | NEW | ~150 |
| `internal/memory/er1.go` | 3 | NEW | ~200 |
| `internal/memory/er1_test.go` | 3 | NEW | ~120 |
| `internal/memory/lifecycle.go` | 4 | EDIT | ~5 |
| `internal/config/config.go` | 5 | EDIT | ~30 |
| `internal/timeline/schema.go` | 1,2 | EDIT | ~20 |
| `internal/agent/loop.go` | 6 | EDIT | ~40 |
| `internal/agent/context.go` | 6 | EDIT | ~30 |
| `cmd/gomikrobot/cmd/gateway.go` | 7 | EDIT | ~40 |

---

## Rollback Path

Each step is additive. Rollback = revert the commit for that step. No existing behavior changes.

- ER1 client: conditional on config (disabled by default)
- Observer: conditional on config (disabled by default)
- Working memory: new table + tool (existing MEMORY.md unchanged)
- Lifecycle policies: additive (new prefixes don't affect existing sources)
