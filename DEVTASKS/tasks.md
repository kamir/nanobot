# GoMikroBot – Migration Tasks (formerly Nanobot)

> Task Planner Agent Output
> Per AGENTS.md Section 3.4

---

## Task Structure

Each task follows:
1. Identify Python module
2. Define Go interface
3. Create tests
4. Implement Go version
5. Parity validation

**Rules:**
- No task > 2 days
- Each has acceptance criteria + parity test + rollback

---

## Phase 1: Core Types & Config

### TASK-001: Initialize Go Module
**Effort:** 0.5 day  
**Priority:** 🔴 Critical  
**Python Source:** `pyproject.toml`

**Acceptance Criteria:**
- [x] Go module initialized at `gomikrobot/` (will rename to `gomikrobot` later)
- [x] Dependencies declared in `go.mod`

**Implementation:**
```bash
cd gomikrobot
go mod init github.com/kamir/gomikrobot
```

**Dependencies to add:**
- `github.com/spf13/cobra` (CLI)
- `github.com/robfig/cron/v3` (scheduler)
- `nhooyr.io/websocket` (WebSocket)
- `github.com/kelseyhightower/envconfig` (env)
- `github.com/go-playground/validator/v10` (validation)
- `github.com/fatih/color` (console)

**Validation:**
```bash
cd gomikrobot && go mod tidy && go build ./...
```

**Parity Test:** N/A (infrastructure)

**Rollback:** Delete `gomikrobot/go.mod`

---

### TASK-002: Implement Config Types
**Effort:** 0.5 day  
**Priority:** 🔴 Critical  
**Python Source:** `nanobot/config/schema.py`

**Acceptance Criteria:**
- [ ] `Config` struct matches Python schema
- [ ] JSON tags with camelCase
- [ ] Default values function

**Implementation:**
Create `gomikrobot/internal/config/config.go` with:
- All config structs from `data-models.md`
- `DefaultConfig()` function

**Validation:**
```bash
cd gomikrobot && go test ./internal/config/...
```

**Test Cases:**
1. Deserialize empty JSON → defaults applied
2. Deserialize full config → all fields populated
3. Environment override works

**Parity Test:**
```bash
# Python
python -c "from nanobot.config.schema import Config; print(Config().model_dump_json(indent=2))"
# Go
./gomikrobot config dump
# Compare output
```

**Rollback:** Delete `internal/config/`

---

### TASK-003: Implement Config Loader
**Effort:** 0.5 day  
**Priority:** 🔴 Critical  
**Python Source:** `nanobot/config/loader.py`

**Acceptance Criteria:**
- [ ] Load from `~/.nanobot/config.json`
- [ ] Create default if missing
- [ ] Environment variable override (NANOBOT_ prefix)

**Implementation:**
Create `gomikrobot/internal/config/loader.go`

**Validation:**
```bash
go test ./internal/config/... -run TestLoader
```

**Test Cases:**
1. Load existing config file
2. Create default when missing
3. Env var overrides file value
4. Invalid JSON returns error

**Parity Test:**
Read Python-created config.json, verify Go parses identically.

**Rollback:** Revert loader.go

---

## Phase 2: LLM Provider

### TASK-004: Implement LLM Provider Interface
**Effort:** 0.5 day  
**Priority:** 🔴 Critical  
**Python Source:** `nanobot/providers/base.py`

**Acceptance Criteria:**
- [ ] `LLMProvider` interface defined
- [ ] `ChatRequest`, `ChatResponse` types
- [ ] `ToolCall`, `ToolDefinition` types

**Implementation:**
Create `gomikrobot/internal/provider/provider.go`

**Validation:**
```bash
go build ./internal/provider/...
```

**Parity Test:** Type definitions only, no runtime behavior.

**Rollback:** Delete `internal/provider/provider.go`

---

### TASK-005: Implement OpenAI-Compatible Client
**Effort:** 1 day  
**Priority:** 🔴 Critical  
**Python Source:** `nanobot/providers/litellm_provider.py`

**Acceptance Criteria:**
- [ ] HTTP client for OpenAI API
- [ ] Model prefix routing (openrouter/, gemini/, etc.)
- [ ] Tool call parsing
- [ ] Error handling

**Implementation:**
Create `gomikrobot/internal/provider/openai.go`

**Validation:**
```bash
go test ./internal/provider/... -run TestOpenAI
```

**Test Cases:**
1. Parse simple response (content only)
2. Parse tool call response
3. Handle API error gracefully
4. Model prefix transformation

**Integration Test (requires API key):**
```bash
OPENROUTER_API_KEY=... go test -tags=integration ./internal/provider/...
```

**Parity Test:**
Send same prompt through Python and Go, compare response structure.

**Rollback:** Use Python provider temporarily

---

## Phase 3: Tool Framework

### TASK-006: Implement Tool Interface & Registry
**Effort:** 0.5 day  
**Priority:** 🔴 Critical  
**Python Source:** `nanobot/agent/tools/base.py`, `registry.py`

**Acceptance Criteria:**
- [ ] `Tool` interface with Name, Description, Parameters, Execute
- [ ] `ToolRegistry` for registration and execution
- [ ] Parameter validation

**Implementation:**
- `gomikrobot/internal/tools/tool.go`
- `gomikrobot/internal/tools/registry.go`

**Validation:**
```bash
go test ./internal/tools/... -run TestRegistry
```

**Test Cases:**
1. Register tool, retrieve by name
2. Execute tool, get result
3. Missing tool returns error
4. Parameter validation errors

**Parity Test:** N/A (infrastructure)

**Rollback:** Delete `internal/tools/`

---

### TASK-007: Implement Filesystem Tools
**Effort:** 0.5 day  
**Priority:** 🔴 Critical  
**Python Source:** `nanobot/agent/tools/filesystem.py`

**Acceptance Criteria:**
- [ ] `read_file` tool
- [ ] `write_file` tool
- [ ] `edit_file` tool
- [ ] `list_dir` tool

**Implementation:**
Create `gomikrobot/internal/tools/filesystem.go`

**Validation:**
```bash
go test ./internal/tools/... -run TestFilesystem
```

**Test Cases:**
1. Read existing file
2. Read missing file → error message
3. Write new file, verify content
4. Edit file, verify replacement
5. List directory contents

**Parity Test:**
```bash
# Python
python -c "import asyncio; from nanobot.agent.tools.filesystem import ReadFileTool; print(asyncio.run(ReadFileTool().execute(path='README.md')))"
# Go
./gomikrobot tool exec read_file --path=README.md
# Compare
```

**Rollback:** Remove filesystem tools from registry

---

### TASK-008: Implement Shell Exec Tool
**Effort:** 0.5 day  
**Priority:** 🔴 Critical  
**Python Source:** `nanobot/agent/tools/shell.py`

**Acceptance Criteria:**
- [ ] Execute shell commands
- [ ] Timeout handling
- [ ] Deny pattern blocklist
- [ ] Optional workspace restriction

**Implementation:**
Create `gomikrobot/internal/tools/shell.go`

**Validation:**
```bash
go test ./internal/tools/... -run TestShell
```

**Test Cases:**
1. Execute `echo hello` → "hello"
2. Timeout after configured seconds
3. Block `rm -rf` → error message
4. Block path traversal when restricted

**Parity Test:**
Execute same commands through Python and Go, compare output.

**Rollback:** Remove exec tool from registry

---

### TASK-009: Implement Web Tools
**Effort:** 0.5 day  
**Priority:** 🟠 High  
**Python Source:** `nanobot/agent/tools/web.py`

**Acceptance Criteria:**
- [ ] `web_search` (Brave API)
- [ ] `web_fetch` (HTTP GET, HTML→text)

**Implementation:**
Create `gomikrobot/internal/tools/web.go`

**Validation:**
```bash
go test ./internal/tools/... -run TestWeb
```

**Test Cases:**
1. Search with mock API response
2. Fetch URL, extract text

**Integration Test:**
```bash
BRAVE_API_KEY=... go test -tags=integration ./internal/tools/... -run TestWebSearch
```

**Rollback:** Remove web tools from registry

---

## Phase 4: Session Manager

### TASK-010: Implement Session Types
**Effort:** 0.25 day  
**Priority:** 🟠 High  
**Python Source:** `nanobot/session/manager.py`

**Acceptance Criteria:**
- [ ] `Session` struct with messages
- [ ] AddMessage, GetHistory, Clear methods

**Implementation:**
Create `gomikrobot/internal/session/session.go`

**Validation:**
```bash
go test ./internal/session/... -run TestSession
```

**Parity Test:** N/A (data structure)

---

### TASK-011: Implement Session Manager
**Effort:** 0.25 day  
**Priority:** 🟠 High  
**Python Source:** `nanobot/session/manager.py`

**Acceptance Criteria:**
- [ ] JSONL persistence
- [ ] GetOrCreate, Save, Delete, List
- [ ] In-memory cache

**Implementation:**
Create `gomikrobot/internal/session/manager.go`

**Validation:**
```bash
go test ./internal/session/... -run TestManager
```

**Test Cases:**
1. Create new session, save, reload
2. Delete session, verify file removed
3. List all sessions

**Parity Test:**
Read Python-created session files with Go.

---

## Phase 5: Message Bus

### TASK-012: Implement Message Bus
**Effort:** 0.5 day  
**Priority:** 🟠 High  
**Python Source:** `nanobot/bus/queue.py`

**Acceptance Criteria:**
- [ ] Channel-based inbound/outbound queues
- [ ] Subscribe by channel name
- [ ] Dispatch outbound to subscribers

**Implementation:**
- `gomikrobot/internal/bus/events.go`
- `gomikrobot/internal/bus/bus.go`

**Validation:**
```bash
go test ./internal/bus/... -run TestBus
```

**Test Cases:**
1. Publish inbound, consume
2. Subscribe outbound, receive
3. Stop cancels blocking consume

---

## Phase 6: Agent Loop

### TASK-013: Implement Agent Loop
**Effort:** 2 days  
**Priority:** 🔴 Critical  
**Python Source:** `nanobot/agent/loop.py`

**Acceptance Criteria:**
- [ ] Consume from bus, process, publish response
- [ ] Loop until stop condition

**Implementation:**
- `gomikrobot/internal/agent/loop.go`

**Validation:**
```bash
go test ./internal/agent/... -run TestLoop
```

---

### TASK-013b: Implement Context Builder (The Soul)
**Effort:** 1 day
**Priority:** 🔴 Critical
**Python Source:** `nanobot/agent/context.py`

**Acceptance Criteria:**
- [x] Load `AGENTS.md`, `SOUL.md`, `USER.md` from workspace
- [x] inject runtime info (OS, time, workspace path)
- [x] Assemble system prompt dynamically

**Implementation:**
Create `gomikrobot/internal/agent/context.go`

**Validation:**
```bash
go test ./internal/agent/... -run TestContextBuilder
```

**Test Cases:**
1. Simple message → LLM response
2. Tool call → execute → return to LLM
3. Max iterations limit
4. Error handling

**Integration Test:**
```bash
OPENROUTER_API_KEY=... go test -tags=integration ./internal/agent/...
```

**Parity Test:**
Same conversation through Python and Go, compare final response.

---

## Phase 7: Channels

### TASK-014: Implement Telegram Channel
**Effort:** 1 day  
**Priority:** 🟠 High  
**Python Source:** `nanobot/channels/telegram.py`

**Acceptance Criteria:**
- [ ] Long polling client
- [ ] Message receiving
- [ ] Message sending with HTML formatting
- [ ] Allowlist check

**Implementation:**
Create `gomikrobot/internal/channels/telegram.go`

**Validation:**
```bash
go test ./internal/channels/... -run TestTelegram
```

**Manual Test:**
1. Configure token in config
2. Run gateway
3. Send message via Telegram
4. Verify response

---

### TASK-015: Implement Discord Channel
**Effort:** 1 day  
**Priority:** 🟠 High  
**Python Source:** `nanobot/channels/discord.py`

**Acceptance Criteria:**
- [ ] WebSocket gateway connection
- [ ] Intent handling
- [ ] Message receiving/sending

**Implementation:**
Create `gomikrobot/internal/channels/discord.go`

**Validation:**
```bash
go test ./internal/channels/... -run TestDiscord
```

---

### TASK-015b: Implement WhatsApp Channel (Native)
**Effort:** 1.5 days
**Priority:** 🟠 High
**Python Source:** `nanobot/channels/whatsapp.py` + `bridge/`
**New approach:** Use `mauidpv/whatsmeow` library.

**Acceptance Criteria:**
- [x] Native Go implementation (no Node.js bridge)
- [x] QR code login via CLI
- [x] Message sending/receiving

**Implementation:**
Create `gomikrobot/internal/channels/whatsapp.go`

**Validation:**
```bash
go test ./internal/channels/... -run TestWhatsApp
```

---

## Phase 8: CLI & Cron

### TASK-016: Implement CLI Framework (Rebranded)
**Effort:** 0.5 day  
**Priority:** 🟡 Medium  
**Python Source:** `nanobot/cli/commands.py`

**Acceptance Criteria:**
- [x] Cobra CLI setup as `gomikrobot`
- [x] Config loader uses `~/.gomikrobot`
- [x] Env Prefix `MIKROBOT_`
- [x] Commands: onboard, status, agent, gateway

**Implementation:**
- `gomikrobot/cmd/nanobot/main.go` -> rename imports/usage
- `gomikrobot/cmd/nanobot/cmd/*.go` -> update help text

**Validation:**
```bash
go build -o gomikrobot ./cmd/nanobot
./gomikrobot --help
./gomikrobot status
```

---

### TASK-017: Implement Cron Service
**Effort:** 0.5 day  
**Priority:** 🟡 Medium  
**Python Source:** `nanobot/cron/service.py`

**Acceptance Criteria:**
- [ ] Job storage (JSON file)
- [ ] Cron expression scheduling
- [ ] Interval scheduling
- [ ] Job execution callback

**Implementation:**
- `gomikrobot/internal/cron/types.go`
- `gomikrobot/internal/cron/service.go`

**Validation:**
```bash
go test ./internal/cron/... -run TestCron
```

**Test Cases:**
1. Add job, verify in list
2. Cron triggers at scheduled time
3. One-shot job deletes after run

---

## Summary

| Task | Description | Effort | Status |
|------|-------------|--------|--------|
| TASK-001 | Init Go module | 0.5d | [ ] |
| TASK-002 | Config types | 0.5d | [ ] |
| TASK-003 | Config loader | 0.5d | [ ] |
| TASK-004 | LLM interface | 0.5d | [ ] |
| TASK-005 | OpenAI client | 1d | [ ] |
| TASK-006 | Tool registry | 0.5d | [ ] |
| TASK-007 | Filesystem tools | 0.5d | [ ] |
| TASK-008 | Shell exec | 0.5d | [ ] |
| TASK-009 | Web tools | 0.5d | [ ] |
| TASK-010 | Session types | 0.25d | [ ] |
| TASK-011 | Session manager | 0.25d | [ ] |
| TASK-012 | Message bus | 0.5d | [ ] |
| TASK-013 | Agent loop | 2d | [x] |
| TASK-013b| Context Builder | 1d | [x] |
| TASK-014 | Telegram | 1d | [ ] (Deferred) |
| TASK-015 | Discord | 1d | [ ] (Deferred) |
| TASK-015b| WhatsApp (Native) | 1.5d | [x] |
| TASK-016 | CLI (Rebranded) | 0.5d | [x] |
| TASK-017 | Cron service | 0.5d | [ ] |

**Total Estimated Effort:** ~13 days

## Phase 9: Timeline & Memory (QMD)

### TASK-QMD-001: SQLite Timeline Migration
**Effort:** 0.5 day  
**Priority:** 🟠 High  

**Acceptance Criteria:**
- [ ] Create `migrations/` folder or embedded SQL.
- [ ] Add `timeline` table with: id, event_id, timestamp, sender, type, content, media_path, vector_id.
- [ ] Ensure indexes on timestamp and sender.

**Implementation:**
- `gomikrobot/internal/channels/whatsapp.go` (integration point)
- `gomikrobot/internal/timeline/schema.go`

---

### TASK-QMD-002: Timeline Service Implementation
**Effort:** 1 day  
**Priority:** 🟠 High  

**Acceptance Criteria:**
- [ ] Service struct with SQLite connection.
- [ ] `AddEvent(event TimelineEvent) error`
- [ ] `GetEvents(filter FilterArgs) ([]TimelineEvent, error)`
- [ ] Support filtering by SenderID and Timestamp range.

**Implementation:**
- `gomikrobot/internal/timeline/service.go`

---

### TASK-QMD-003: Qdrant Vector Integration
**Effort:** 1.5 days  
**Priority:** 🟠 High  

**Acceptance Criteria:**
- [ ] Helper for Qdrant HTTP/gRPC API.
- [ ] Function to generate embeddings (using `provider.Embed` if available, or dedicated call).
- [ ] `StoreMemory(content string, meta map[string]interface{})`
- [ ] `SearchMemory(query string) []Result`

**Implementation:**
- `gomikrobot/internal/memory/qdrant.go`
- `gomikrobot/internal/memory/vector.go`

---

### TASK-QMD-004: REST API for Timeline
**Effort:** 0.5 day  
**Priority:** 🟡 Medium  

**Acceptance Criteria:**
- [ ] `GET /api/v1/timeline` -> standard JSON response.
- [ ] `GET /api/v1/timeline/:id/media` -> serve local file safely.
- [ ] Auth/Authz middleware (basic or token).

**Implementation:**
- `gomikrobot/cmd/gomikrobot/server/api.go`

---

### TASK-SPA-001: Timeline SPA Frontend
**Effort:** 2 days  
**Priority:** 🟡 Medium  

**Acceptance Criteria:**
- [ ] Single HTML file `timeline.html` served by Gateway.
- [ ] Vue.js or similar reactive UI.
- [ ] Vertical timeline component.
- [ ] Filter dropdown by UserID.
- [ ] Opacity logic: Selected user = 1.0, Others = 0.3.
- [ ] Audio player integration.

**Implementation:**
- `gomikrobot/web/timeline.html`
- `gomikrobot/web/app.js`

---

### Summary (Updated)

| Task | Description | Effort | Status |
|------|-------------|--------|--------|
| TASK-QMD-001 | SQLite Timeline | 0.5d | [ ] |
| TASK-QMD-002 | Timeline Service | 1d | [ ] |
| TASK-QMD-003 | Qdrant Integration | 1.5d | [ ] |
| TASK-QMD-004 | REST API | 0.5d | [ ] |
| TASK-SPA-001 | Timeline SPA | 2d | [ ] |
