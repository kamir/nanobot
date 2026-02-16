# Memory Architecture v2: Design Specification

> Version: 2.0 | Date: 2026-02-16 | Status: Proposed

---

## 1. Goals

1. **ER1 integration**: Index personal memories from the ER1 capture service into semantic search
2. **Observational memory**: LLM-compressed conversation logs for long-session continuity (inspired by Mastra)
3. **Scoped working memory**: Per-user and per-thread persistent scratchpads replacing global `MEMORY.md`
4. **Zero regression**: All 33 existing memory tests continue passing; existing 4-layer architecture unchanged

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     MEMORY ARCHITECTURE v2                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  SOURCES (content producers)                                    │
│  ├─ Soul Files        (AGENTS.md, SOUL.md, etc.)  → soul:*     │
│  ├─ Conversations     (auto-indexed Q&A pairs)    → conversation:* │
│  ├─ Tool Results      (auto-indexed exec output)  → tool:*     │
│  ├─ Group Sharing     (Kafka/S3 artifacts)        → group:*    │
│  ├─ ER1 Sync          (personal memory transcripts) → er1:*    │  NEW
│  └─ Observer          (LLM-compressed observations) → observation:* │  NEW
│                                                                 │
│  STORAGE                                                        │
│  ├─ VectorStore       (SQLite-vec / Qdrant)       embeddings   │
│  ├─ Timeline DB       (SQLite)                    structured   │
│  └─ WorkingMemory     (SQLite, per-user/thread)   scratchpad   │  NEW
│                                                                 │
│  RETRIEVAL (context assembly)                                   │
│  ├─ RAG Injection     (vector search → system prompt)          │
│  ├─ Working Memory    (scoped scratchpad → system prompt)      │  NEW
│  └─ Observation Log   (compressed history → system prompt)     │  NEW
│                                                                 │
│  LIFECYCLE                                                      │
│  ├─ TTL Pruning       (per-source retention policies)          │
│  ├─ Observer Trigger  (token threshold → compress)             │  NEW
│  ├─ Reflector Trigger (observation overflow → consolidate)     │  NEW
│  └─ Expertise Tracker (skill proficiency scoring)              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. New Component: ER1 Client

### Purpose
Sync personal memories from the ER1 service into GoMikroBot's vector store as `er1:` source chunks.

### Design

```go
// internal/memory/er1.go

type ER1Config struct {
    BaseURL    string        // e.g. "http://127.0.0.1:8080"
    APIKey     string        // X-API-KEY header
    UserID     string        // uid for /user/access
    SyncInterval time.Duration // default: 5 minutes
}

type ER1Client struct {
    config     ER1Config
    httpClient *http.Client
    ctxID      string        // obtained from /user/access
    service    *MemoryService
    lastSync   time.Time
}
```

### Sync Flow

```
1. Authenticate: POST /user/access → obtain ctx_id
2. Fetch memories: GET /memory/{ctx_id}?startDate={lastSync}
3. For each memory with a non-empty transcript:
   a. Format: "ER1 Memory [{tags}] at {location}: {transcript}"
   b. Store via MemoryService.Store(content, "er1:{memory_id}", tags)
4. Update lastSync timestamp
5. Repeat on SyncInterval timer
```

### Lifecycle
- Source prefix: `er1:`
- TTL: 0 (permanent — ER1 is the authoritative source, local copy for search)
- Pruning: Controlled by ER1 (if memory deleted upstream, local copy remains until re-sync detects absence)

### Configuration
- Env: `MIKROBOT_ER1_URL`, `MIKROBOT_ER1_API_KEY`, `MIKROBOT_ER1_USER_ID`
- Config file: `er1.url`, `er1.api_key`, `er1.user_id`
- Graceful degradation: If ER1 is unreachable, log warning and continue without ER1 memories

---

## 4. New Component: Observational Memory

### Purpose
Compress old conversation history into prioritized, date-anchored observation notes using an LLM background pass. Prevents context window degradation in long sessions.

### Design

```go
// internal/memory/observer.go

type ObserverConfig struct {
    Enabled           bool
    Model             string        // LLM model for compression (default: same as agent)
    MessageThreshold  int           // compress after N unobserved messages (default: 50)
    MaxObservations   int           // trigger reflector after this many (default: 200)
}

type Observer struct {
    config    ObserverConfig
    provider  provider.LLMProvider
    service   *MemoryService
    timeline  TimelineReader        // interface to read conversation history
}

type Observation struct {
    ID            string
    Content       string     // compressed observation text
    ObservedAt    time.Time  // when observation was created
    ReferencedAt  time.Time  // earliest event referenced
    Priority      string     // "high", "medium", "low"
    SessionID     string     // which session this covers
    MessageRange  [2]int     // [startIdx, endIdx] of observed messages
}
```

### Observation Flow

```
1. Trigger: After agent response, check unobserved message count for session
2. If count >= MessageThreshold:
   a. Collect unobserved messages (oldest first)
   b. Send to LLM with observer prompt:
      "Compress these conversation messages into dated observation notes.
       For each observation:
       - Priority: high/medium/low
       - Referenced date: when the event occurred
       - Content: 1-2 sentence factual summary
       Group by date. Focus on facts, decisions, and user preferences."
   c. Parse response into Observation structs
   d. Store each observation:
      - In vector store: source="observation:{session_id}", for RAG
      - In timeline DB: structured record for ordered retrieval
   e. Mark messages as observed
3. Non-blocking: runs in background goroutine after response
```

### Reflector Flow

```
1. Trigger: observation count for session exceeds MaxObservations
2. Collect all observations for session
3. Send to LLM with reflector prompt:
   "Consolidate these observations. Combine related items, remove
    superseded information, identify patterns. Preserve all high-priority
    items. Target: reduce to 60% of original count."
4. Replace old observations with consolidated set
5. Re-index consolidated observations in vector store
```

### Context Injection

Observations are injected into the system prompt as a separate section:

```markdown
# Observations (Session Memory)
## 2026-02-15
- [HIGH] User prefers Go over Python for new projects
- [MED] Working on deployment pipeline for GoMikroBot
## 2026-02-16
- [HIGH] Memory architecture v2 design in progress
- [LOW] User mentioned upcoming vacation next week
```

This section is placed between the bootstrap/identity section and the RAG results, providing a stable prefix for prompt caching.

### Lifecycle
- Source prefix: `observation:`
- TTL: 0 (permanent — observations are already compressed, pruning handled by reflector)
- Retention policy added to `DefaultPolicies()`

---

## 5. New Component: Scoped Working Memory

### Purpose
Replace the global `MEMORY.md` file with a per-user, per-thread persistent scratchpad. Enables the agent to maintain structured knowledge about each user across conversations.

### Design

```go
// internal/memory/working.go

type WorkingMemoryConfig struct {
    Enabled  bool
    Template string   // default markdown template
}

type WorkingMemoryStore struct {
    db *sql.DB
}

type WorkingMemoryEntry struct {
    ResourceID string    // user identifier (e.g., WhatsApp phone number)
    ThreadID   string    // conversation/session ID (empty = resource-scoped)
    Content    string    // markdown content
    UpdatedAt  time.Time
}
```

### Schema

```sql
CREATE TABLE IF NOT EXISTS working_memory (
    resource_id TEXT NOT NULL,
    thread_id   TEXT NOT NULL DEFAULT '',
    content     TEXT NOT NULL DEFAULT '',
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (resource_id, thread_id)
);
```

### Operations

```go
// Load returns working memory for a resource, optionally thread-scoped.
// Falls back to resource-level if thread-specific doesn't exist.
func (w *WorkingMemoryStore) Load(resourceID, threadID string) (string, error)

// Save persists working memory. Merge semantics: the agent provides
// a complete replacement of the content (not a diff).
func (w *WorkingMemoryStore) Save(resourceID, threadID, content string) error
```

### Agent Tool

```go
// tools/working_memory.go
// Name: "update_working_memory"
// Description: "Update your persistent notes about the current user or conversation.
//              Content should be structured markdown with key facts, preferences,
//              and context you want to remember across sessions."
// Parameters: { "content": string, "scope": "resource"|"thread" }
```

### Context Injection

Working memory is loaded during context building and injected as:

```markdown
# Working Memory
{content from working_memory table for current resource/thread}
```

Placed after identity/bootstrap and before observations.

### Scoping Rules

| Scope | Key | Use Case |
|-------|-----|----------|
| Resource | `(resourceID, "")` | User preferences, name, long-term facts |
| Thread | `(resourceID, threadID)` | Task-specific state, current project context |

**Lookup order**: Thread-specific first, then resource-level. Both are included if both exist.

### Default Template

```markdown
## User Profile
- Name:
- Preferences:
- Language:

## Current Context
- Active tasks:
- Recent topics:

## Notes
```

---

## 6. Retention Policy Updates

```go
func DefaultPolicies() []RetentionPolicy {
    return []RetentionPolicy{
        {SourcePrefix: "soul:",          TTL: 0},                        // permanent
        {SourcePrefix: "user",           TTL: 0},                        // permanent
        {SourcePrefix: "consolidated:",  TTL: 0},                        // permanent
        {SourcePrefix: "observation:",   TTL: 0},                        // permanent (NEW)
        {SourcePrefix: "er1:",           TTL: 0},                        // permanent (NEW)
        {SourcePrefix: "conversation:",  TTL: 30 * 24 * time.Hour},     // 30 days
        {SourcePrefix: "tool:",          TTL: 14 * 24 * time.Hour},     // 14 days
        {SourcePrefix: "group:",         TTL: 60 * 24 * time.Hour},     // 60 days
    }
}
```

---

## 7. Context Assembly Order (Updated)

```
System Prompt Assembly:
  1. Identity         (runtime info, version, date)
  2. Bootstrap Files  (AGENTS.md, SOUL.md, USER.md, TOOLS.md, IDENTITY.md)
  3. Working Memory   (scoped per user/thread)              ← NEW
  4. Observations     (compressed session history)           ← NEW
  5. Skills Summary   (registered tools + skill docs)
  6. RAG Context      (vector search: soul + conversation + tool + group + er1 + observation)
  7. Conversation     (recent message history)
```

Sections 1-4 form a **stable prefix** that rarely changes within a session, enabling prompt caching.

---

## 8. Gateway Wiring (Updated)

```
gateway.go initialization order:

  4c. Memory System (existing)
      ├─ VectorStore (SQLite-vec, 1536 dims)
      └─ MemoryService(store, embedder)

  4c-ii. Working Memory Store (NEW)
      └─ WorkingMemoryStore(timeSvc.DB())

  4c-iii. ER1 Client (NEW, conditional)
      ├─ ER1Client(config, httpClient, memorySvc)
      └─ Start sync goroutine if ER1 configured

  4c-iv. Observer (NEW, conditional)
      └─ Observer(config, provider, memorySvc, timeSvc)

  5. Auto-Indexer (existing)
  5a. Expertise Tracker (existing)
  5b. Loop (updated — add WorkingMemory, Observer to LoopOptions)
  5b. Soul file indexing (existing)

  Start goroutines:
  - autoIndexer.Run(ctx)           (existing)
  - er1Client.SyncLoop(ctx)        (NEW)
  - lifecycleMgr daily ticker      (existing)
```

---

## 9. New Files Summary

| File | Package | Purpose |
|------|---------|---------|
| `internal/memory/er1.go` | memory | ER1 HTTP client + sync loop |
| `internal/memory/er1_test.go` | memory | ER1 client tests (mock HTTP) |
| `internal/memory/observer.go` | memory | Observer + Reflector (LLM compression) |
| `internal/memory/observer_test.go` | memory | Observer tests (mock LLM) |
| `internal/memory/working.go` | memory | Scoped working memory store |
| `internal/memory/working_test.go` | memory | Working memory tests |

### Modified Files

| File | Change |
|------|--------|
| `internal/memory/lifecycle.go` | Add `er1:` and `observation:` to DefaultPolicies |
| `internal/agent/loop.go` | Add WorkingMemory + Observer to LoopOptions, inject into context |
| `internal/agent/context.go` | Load working memory + observations into system prompt |
| `cmd/gomikrobot/cmd/gateway.go` | Initialize and wire new components |
| `internal/config/config.go` | Add ER1Config and ObserverConfig fields |
| `internal/timeline/schema.go` | Add working_memory and observations tables |

---

## 10. Acceptance Criteria

1. `go test ./internal/memory/...` — all tests pass (existing 33 + new)
2. `go build ./cmd/gomikrobot` — compiles without errors
3. Gateway starts with ER1 disabled — no errors, no ER1 log noise
4. Gateway starts with ER1 enabled — syncs memories, searchable via `recall`
5. Long conversation (50+ messages) — observer compresses, observations appear in context
6. Multi-user WhatsApp — each user gets separate working memory
7. Working memory persists across gateway restarts
8. Observations are searchable via RAG (vector store)
