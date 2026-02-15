# GoMikroBot Administration Guide

This document provides comprehensive administrative reference for deploying, configuring, securing, and operating GoMikroBot. All details are derived directly from the source code.

---

## Table of Contents

1. [Configuration Reference](#1-configuration-reference)
2. [Security Model](#2-security-model)
3. [LLM Provider Configuration](#3-llm-provider-configuration)
4. [Memory and RAG Administration](#4-memory-and-rag-administration)
5. [Token Quota Management](#5-token-quota-management)
6. [Extending GoMikroBot](#6-extending-gomikrobot)
7. [Runtime Settings](#7-runtime-settings)
8. [Web User Management](#8-web-user-management)
9. [Audit and Compliance](#9-audit-and-compliance)

---

## 1. Configuration Reference

### Config File Location

```
~/.gomikrobot/config.json
```

The config file is created by `gomikrobot onboard` and persisted by `config.Save()`. The directory is created with `0700` permissions; the file is written with `0600` permissions.

### Loading Order

Configuration values are resolved in the following precedence (highest wins):

1. **Environment variables** (prefix: `MIKROBOT_`)
2. **Config file** (`~/.gomikrobot/config.json`)
3. **Built-in defaults** (from `DefaultConfig()`)

### Root Config Struct

```go
type Config struct {
    Agents    AgentsConfig    `json:"agents"`
    Channels  ChannelsConfig  `json:"channels"`
    Providers ProvidersConfig `json:"providers"`
    Gateway   GatewayConfig   `json:"gateway"`
    Tools     ToolsConfig     `json:"tools"`
}
```

### Agent Configuration

```go
type AgentsConfig struct {
    Defaults AgentDefaults `json:"defaults"`
}

type AgentDefaults struct {
    Workspace         string  `json:"workspace"         envconfig:"WORKSPACE"`
    WorkRepoPath      string  `json:"workRepoPath"      envconfig:"WORK_REPO_PATH"`
    SystemRepoPath    string  `json:"systemRepoPath"    envconfig:"SYSTEM_REPO_PATH"`
    Model             string  `json:"model"             envconfig:"MODEL"`
    MaxTokens         int     `json:"maxTokens"         envconfig:"MAX_TOKENS"`
    Temperature       float64 `json:"temperature"       envconfig:"TEMPERATURE"`
    MaxToolIterations int     `json:"maxToolIterations"  envconfig:"MAX_TOOL_ITERATIONS"`
}
```

| Field | Default | Description |
|---|---|---|
| `Workspace` | `~/GoMikroBot-Workspace` | Directory for soul files, session state, and agent workspace. Forced to `~/GoMikroBot-Workspace` at load time. |
| `WorkRepoPath` | `~/GoMikroBot-Workspace` | Root directory for filesystem write operations. |
| `SystemRepoPath` | *(machine-specific)* | Path to the system/identity repository (contains Day2Day tasks, soul files). |
| `Model` | `anthropic/claude-sonnet-4-5` | Default LLM model identifier. |
| `MaxTokens` | `8192` | Maximum tokens per LLM response. |
| `Temperature` | `0.7` | LLM sampling temperature. |
| `MaxToolIterations` | `20` | Maximum agentic tool-call loop iterations per message. |

### Provider Configuration

```go
type ProvidersConfig struct {
    Anthropic    ProviderConfig     `json:"anthropic"`
    OpenAI       ProviderConfig     `json:"openai"`
    LocalWhisper LocalWhisperConfig `json:"localWhisper"`
    OpenRouter   ProviderConfig     `json:"openrouter"`
    DeepSeek     ProviderConfig     `json:"deepseek"`
    Groq         ProviderConfig     `json:"groq"`
    Gemini       ProviderConfig     `json:"gemini"`
    VLLM         ProviderConfig     `json:"vllm"`
}

type ProviderConfig struct {
    APIKey  string `json:"apiKey"             envconfig:"API_KEY"`
    APIBase string `json:"apiBase,omitempty"  envconfig:"API_BASE"`
}

type LocalWhisperConfig struct {
    Enabled    bool   `json:"enabled"    envconfig:"WHISPER_ENABLED"`
    Model      string `json:"model"      envconfig:"WHISPER_MODEL"`
    BinaryPath string `json:"binaryPath" envconfig:"WHISPER_BINARY_PATH"`
}
```

| Provider | Default API Base | Notes |
|---|---|---|
| OpenAI | `https://api.openai.com/v1` | Used when `apiBase` is empty |
| OpenRouter | `https://openrouter.ai/api/v1` | OpenAI-compatible API format |
| LocalWhisper | N/A | Local binary at `/opt/homebrew/bin/whisper`, model `base` |

### Channel Configuration

```go
type ChannelsConfig struct {
    Telegram TelegramConfig `json:"telegram"`
    Discord  DiscordConfig  `json:"discord"`
    WhatsApp WhatsAppConfig `json:"whatsapp"`
    Feishu   FeishuConfig   `json:"feishu"`
}
```

Each channel config follows the same pattern:

| Field | Type | Description |
|---|---|---|
| `Enabled` | `bool` | Whether the channel is active |
| `Token` / `AppID` | `string` | Authentication credential |
| `AllowFrom` | `[]string` | Allowlist of sender identifiers |

WhatsApp-specific fields:

| Field | Env Var | Default | Description |
|---|---|---|---|
| `DropUnauthorized` | `MIKROBOT_CHANNELS_WHATSAPP_DROP_UNAUTHORIZED` | `false` | Silently drop messages from unknown senders |
| `IgnoreReactions` | `MIKROBOT_CHANNELS_WHATSAPP_IGNORE_REACTIONS` | `false` | Ignore reaction messages |

### Gateway Configuration

```go
type GatewayConfig struct {
    Host          string `json:"host"          envconfig:"HOST"`
    Port          int    `json:"port"          envconfig:"PORT"`
    DashboardPort int    `json:"dashboardPort" envconfig:"DASHBOARD_PORT"`
}
```

| Field | Default | Description |
|---|---|---|
| `Host` | `127.0.0.1` | Bind address (localhost only by default for security) |
| `Port` | `18790` | API port |
| `DashboardPort` | `18791` | Dashboard/Web UI port |

### Tools Configuration

```go
type ToolsConfig struct {
    Exec ExecToolConfig `json:"exec"`
    Web  WebToolConfig  `json:"web"`
}

type ExecToolConfig struct {
    Timeout             time.Duration `json:"timeout"`
    RestrictToWorkspace bool          `json:"restrictToWorkspace" envconfig:"EXEC_RESTRICT_WORKSPACE"`
}

type WebToolConfig struct {
    Search SearchConfig `json:"search"`
}

type SearchConfig struct {
    APIKey     string `json:"apiKey" envconfig:"BRAVE_API_KEY"`
    MaxResults int    `json:"maxResults"`
}
```

| Field | Default | Description |
|---|---|---|
| `Exec.Timeout` | `60s` | Shell command timeout |
| `Exec.RestrictToWorkspace` | `true` | Confine shell execution to workspace/work-repo paths |
| `Web.Search.MaxResults` | `10` | Maximum web search results |

### Key Environment Variables

| Variable | Env Prefix | Description |
|---|---|---|
| `OPENAI_API_KEY` | (direct fallback) | OpenAI API key; also used as fallback if no provider key is set |
| `OPENROUTER_API_KEY` | (direct fallback) | OpenRouter API key; used as secondary fallback |
| `MIKROBOT_AGENTS_WORKSPACE` | `MIKROBOT_AGENTS` | Workspace directory path |
| `MIKROBOT_AGENTS_WORK_REPO_PATH` | `MIKROBOT_AGENTS` | Work repository path |
| `MIKROBOT_AGENTS_SYSTEM_REPO_PATH` | `MIKROBOT_AGENTS` | System/identity repository path |
| `MIKROBOT_AGENTS_MODEL` | `MIKROBOT_AGENTS` | Default LLM model |
| `MIKROBOT_AGENTS_MAX_TOKENS` | `MIKROBOT_AGENTS` | Maximum tokens per response |
| `MIKROBOT_AGENTS_MAX_TOOL_ITERATIONS` | `MIKROBOT_AGENTS` | Max tool-call loop iterations |
| `MIKROBOT_GATEWAY_HOST` | `MIKROBOT_GATEWAY` | Gateway bind address |
| `MIKROBOT_GATEWAY_PORT` | `MIKROBOT_GATEWAY` | API port |
| `MIKROBOT_GATEWAY_DASHBOARD_PORT` | `MIKROBOT_GATEWAY` | Dashboard port |
| `MIKROBOT_TOOLS_EXEC_RESTRICT_WORKSPACE` | `MIKROBOT_TOOLS_EXEC` | Restrict shell to workspace |

---

## 2. Security Model

GoMikroBot implements defense in depth across multiple layers: tool tiering, policy evaluation, shell filtering, filesystem confinement, and attack intent detection.

### Tool Risk Tiers

Every tool implements the `Tool` interface. Tools that also implement `TieredTool` declare their risk tier. Unclassified tools default to Tier 0.

```go
const (
    TierReadOnly  = 0  // Read-only internal tools
    TierWrite     = 1  // Controlled write/internal effects
    TierHighRisk  = 2  // External or high-impact actions
)
```

| Tier | Tools | Description |
|---|---|---|
| 0 (ReadOnly) | `read_file`, `list_dir`, `resolve_path`, `recall` | Always allowed by policy |
| 1 (Write) | `write_file`, `edit_file`, `remember` | Allowed by default policy (MaxAutoTier=1) |
| 2 (HighRisk) | `exec` | Denied by default policy; requires MaxAutoTier >= 2 |

### Policy Engine

The policy engine evaluates every tool invocation before execution.

**Interface:**

```go
type Engine interface {
    Evaluate(ctx Context) Decision
}

type Context struct {
    Sender    string
    Channel   string
    Tool      string
    Tier      int
    Arguments map[string]any
    TraceID   string
}

type Decision struct {
    Allow   bool
    Reason  string
    Tier    int
    Ts      time.Time
    TraceID string
}
```

**DefaultEngine behavior:**

1. Tier 0 tools are always allowed (`tier_0_always_allowed`).
2. If `AllowedSenders` is configured and the sender is not in the set, the request is denied (`sender_not_authorized`).
3. If the tool's tier exceeds `MaxAutoTier` (default: 1), the request is denied (`tier_N_requires_approval`).
4. Otherwise, the request is allowed (`tier_N_auto_approved`).

Every policy decision is logged to the `policy_decisions` table in the timeline database (see [Audit and Compliance](#9-audit-and-compliance)).

### Shell Security (exec tool)

The shell execution tool (`internal/tools/shell.go`) implements multiple protection layers:

**Strict allow-list mode** (enabled by default, `StrictAllowList: true`):

Only commands matching allow patterns are permitted:

```
git, ls, cat, pwd, rg, grep, sed, head, tail, wc, echo
```

**Deny pattern filtering:**

Even when allow-listed, commands are checked against deny patterns that block destructive operations:

| Category | Patterns |
|---|---|
| Recursive deletion | `rm -rf`, `rm -r .`, `rm *`, `find -delete`, `unlink`, `rmdir` |
| Version control deletion | `git rm` |
| Device/partition ops | `dd of=/dev/`, `mkfs`, `fdisk`, `format` |
| Device redirect | `> /dev/` |
| Permission escalation | `chmod -R 777`, `chown -R` on root/home |
| Fork bombs | `:(){ :|:& };:` |
| System control | `shutdown`, `reboot`, `halt`, `init [0-6]`, `systemctl start/stop/restart/enable/disable` |

**Path traversal protection:**

When `RestrictToWorkspace` is `true`, commands containing path traversal patterns (`../`, `..\`, `/..`, `\..`) are rejected. Working directory arguments are validated against allowed roots (workspace and work repo paths).

**Timeout:** Default 60 seconds. Commands exceeding the timeout are killed.

### Filesystem Security

File operations (`internal/tools/filesystem.go`) enforce the following rules:

- **read_file** and **list_dir**: Can access any path on the filesystem (Tier 0).
- **write_file** and **edit_file**: Restricted to the work repo root path (Tier 1). Writes outside the work repo return `"Error: path outside work repo."`.
- **Path traversal**: The `../` pattern is blocked by the shell tool. Filesystem tools use `filepath.Rel()` to verify paths are within the work repo.
- **Tilde expansion**: Paths starting with `~` are expanded to the user's home directory.

### Attack Intent Detection

The agent loop (`internal/agent/loop.go`) scans user messages for malicious intent patterns before invoking the LLM:

```
delete repo, repo delete, remove repo, wipe repo, delete content,
delete all files, remove all files, rm -rf, losch repo, losch all,
datei(en) losch
```

When detected, the bot returns a safety response and does not process the message further. The same safety string (`"Ey, du spinnst wohl?"`) is checked in tool execution results as a circuit breaker.

### WhatsApp Authorization

WhatsApp sender authorization uses three lists stored as runtime settings:

| Setting Key | Description |
|---|---|
| `whatsapp_allowlist` | Newline-separated approved WhatsApp JIDs |
| `whatsapp_denylist` | Newline-separated denied JIDs |
| `whatsapp_pending` | Newline-separated JIDs awaiting approval |

Unknown senders are automatically placed in the `pending` list. Administration is performed via:

```bash
gomikrobot whatsapp-auth --approve <JID>
gomikrobot whatsapp-auth --deny <JID>
gomikrobot whatsapp-auth --list
```

A `whatsapp_pair_token` runtime setting supports out-of-band authentication.

---

## 3. LLM Provider Configuration

### Provider Architecture

All LLM providers are accessed through a single `OpenAIProvider` implementation that speaks the OpenAI-compatible API format. This means OpenAI, OpenRouter, Anthropic (via OpenRouter), DeepSeek, Groq, Gemini, and VLLM endpoints can all be used.

**LLMProvider interface:**

```go
type LLMProvider interface {
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    Transcribe(ctx context.Context, req *AudioRequest) (*AudioResponse, error)
    Speak(ctx context.Context, req *TTSRequest) (*TTSResponse, error)
    DefaultModel() string
}
```

**Embedder interface** (optional, used for memory/RAG):

```go
type Embedder interface {
    Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)
}
```

Not all providers implement `Embedder`. Callers check via type assertion: `if emb, ok := prov.(Embedder); ok { ... }`.

### Supported Capabilities

| Capability | API Endpoint | Default Model | Notes |
|---|---|---|---|
| Chat completion | `/chat/completions` | `anthropic/claude-sonnet-4-5` | Supports tool calling via `tool_choice: auto` |
| Audio transcription | `/audio/transcriptions` | `whisper-1` | Whisper API; also supports local whisper binary |
| Text-to-speech | `/audio/speech` | `tts-1` | Default voice: `nova`, output format: `opus` |
| Embeddings | `/embeddings` | `text-embedding-3-small` | Used by memory/RAG system |

### ChatRequest Defaults

| Field | Default | Description |
|---|---|---|
| `MaxTokens` | `4096` | Configured per-call in the agent loop |
| `Temperature` | `0.7` | Sampling temperature |
| HTTP timeout | `120s` | HTTP client timeout for provider calls |

### API Key Fallback Chain

The config loader applies the following fallback for the OpenAI provider API key:

1. `cfg.Providers.OpenAI.APIKey` (from config file or `MIKROBOT_OPENAI_API_KEY` env var)
2. `OPENAI_API_KEY` environment variable
3. `OPENROUTER_API_KEY` environment variable

### Usage Tracking

Every LLM response includes token usage:

```go
type Usage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}
```

Usage is accumulated per task in the `tasks` table (see [Token Quota Management](#5-token-quota-management)).

---

## 4. Memory and RAG Administration

### Architecture Overview

```
User Query --> Embed(query) --> VectorStore.Search(top 5) --> Filter(score >= 0.3)
                                                                    |
                                                          Inject into system prompt
                                                          as "Relevant Memory" section
```

### Components

| Component | File | Description |
|---|---|---|
| `VectorStore` interface | `internal/memory/vector.go` | Pluggable vector storage backend |
| `SQLiteVecStore` | `internal/memory/sqlite_vec.go` | Default SQLite-based store (zero external dependencies) |
| `MemoryService` | `internal/memory/service.go` | High-level Store/Search with automatic embedding |
| `SoulFileIndexer` | `internal/memory/indexer.go` | Indexes soul files by chunking on `##` headers |
| `RememberTool` | `internal/tools/memory.go` | Agent tool for storing user memories (Tier 1) |
| `RecallTool` | `internal/tools/memory.go` | Agent tool for searching memories (Tier 0) |

### VectorStore Interface

```go
type VectorStore interface {
    Upsert(ctx context.Context, id string, vector []float32, payload map[string]interface{}) error
    Search(ctx context.Context, vector []float32, limit int) ([]Result, error)
    EnsureCollection(ctx context.Context) error
}
```

### SQLiteVecStore Details

- Embeddings are stored as BLOBs (little-endian float32 arrays) in the `memory_chunks` table.
- Cosine similarity is computed in Go -- at fewer than 10,000 chunks this is sub-millisecond.
- The table is created by the schema migration in the timeline database.
- No external vector DB dependency required.

### Memory Chunks Table Schema

```sql
CREATE TABLE IF NOT EXISTS memory_chunks (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    embedding BLOB,
    source TEXT NOT NULL DEFAULT 'user',
    tags TEXT DEFAULT '',
    version INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_memory_chunks_source ON memory_chunks(source);
```

### Chunk ID Generation

Chunk IDs are deterministic, derived from the source and content:

```go
func chunkID(source, content string) string {
    h := sha256.Sum256([]byte(source + ":" + content))
    return fmt.Sprintf("%x", h[:8])
}
```

This ensures idempotent indexing -- re-indexing the same content overwrites rather than duplicates.

### Graceful Degradation

If no `Embedder` is available (the provider does not implement the interface), all memory operations are no-ops:

- `Store()` returns `("", nil)` -- no error, no storage.
- `Search()` returns `(nil, nil)` -- no error, no results.
- Memory tools (`remember`, `recall`) are only registered when `MemoryService` is non-nil.

### Soul File Indexing

The `SoulFileIndexer` reads the following files from the workspace directory:

```
AGENTS.md, SOUL.md, USER.md, TOOLS.md, IDENTITY.md
```

Each file is split by `##` headers using `ChunkByHeaders()`. Each chunk is embedded and stored with:

- **Source**: `soul:<filename>` (e.g., `soul:SOUL.md`)
- **Tags**: The `##` heading text
- **ID**: Deterministic hash of source + content

Errors on individual files are logged but do not abort overall indexing.

### RAG Pipeline

When processing a user message, the agent loop:

1. Calls `MemoryService.Search(query, 5)` to retrieve the top 5 similar chunks.
2. Filters results by relevance score >= 0.3 (30%).
3. Appends matching chunks to the system prompt (first message) under a `# Relevant Memory` section.
4. Format: `- [source={source}, relevance={score}%] {content}`

### Default Embedding Model

The `OpenAIProvider.Embed()` method defaults to `text-embedding-3-small` when no model is specified in the request.

---

## 5. Token Quota Management

### Overview

GoMikroBot tracks token usage per task and enforces a configurable daily token budget.

### Per-Task Token Tracking

Every LLM call in the agent loop records usage via `UpdateTaskTokens()`:

```go
func (s *TimelineService) UpdateTaskTokens(taskID string, prompt, completion, total int) error
```

Token counts are accumulated additively (a single task may involve multiple LLM calls in the agentic loop).

Fields in the `tasks` table:

| Column | Type | Description |
|---|---|---|
| `prompt_tokens` | `INTEGER` | Total prompt tokens consumed |
| `completion_tokens` | `INTEGER` | Total completion tokens consumed |
| `total_tokens` | `INTEGER` | Sum of prompt + completion tokens |

### Daily Token Limit

The daily token limit is configured as a runtime setting:

```
Key: daily_token_limit
Value: integer (e.g., "100000")
```

**Enforcement logic** (checked before every LLM call):

1. Read `daily_token_limit` from the `settings` table.
2. If not set or not a positive integer, skip quota check (fail-open / unlimited).
3. Call `GetDailyTokenUsage()` which sums `total_tokens` for all tasks created today.
4. If daily usage >= limit, return an error message to the user and do not call the LLM.

```go
func (s *TimelineService) GetDailyTokenUsage() (int, error) {
    // SELECT COALESCE(SUM(total_tokens), 0) FROM tasks WHERE created_at >= date('now')
}
```

### Quota Exceeded Message

When the daily quota is exceeded, the user receives:

```
Daily token quota exceeded (X/Y). Please try again tomorrow or ask an admin to increase the limit.
```

### Setting the Quota

Via the dashboard API:

```
POST /api/v1/settings
Content-Type: application/json

{"key": "daily_token_limit", "value": "100000"}
```

Or via SQLite directly:

```sql
INSERT INTO settings (key, value, updated_at) VALUES ('daily_token_limit', '100000', datetime('now'))
ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at;
```

To remove the quota (unlimited), delete the setting or set it to `0` or empty.

---

## 6. Extending GoMikroBot

### Adding a New Tool

1. Create a new file in `gomikrobot/internal/tools/`.

2. Implement the `Tool` interface:

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]any   // JSON Schema
    Execute(ctx context.Context, params map[string]any) (string, error)
}
```

3. Optionally implement `TieredTool` to declare a risk tier:

```go
type TieredTool interface {
    Tool
    Tier() int  // 0=ReadOnly, 1=Write, 2=HighRisk
}
```

If `TieredTool` is not implemented, the tool defaults to Tier 0 (ReadOnly).

4. Register the tool in `registerDefaultTools()` in `gomikrobot/internal/agent/loop.go`:

```go
func (l *Loop) registerDefaultTools() {
    // ... existing tools ...
    l.registry.Register(NewMyTool())
}
```

5. The tool will automatically appear in LLM tool definitions and be subject to policy evaluation based on its tier.

**Helper functions** available for parameter extraction:

- `GetString(params, key, defaultVal)` -- extract string parameter
- `GetInt(params, key, defaultVal)` -- extract int parameter (handles float64 from JSON)
- `GetBool(params, key, defaultVal)` -- extract bool parameter

### Adding a New Channel

1. Implement the `Channel` interface in `gomikrobot/internal/channels/`.

2. Subscribe to the message bus for outbound messages:

```go
bus.SubscribeOutbound(func(msg *bus.OutboundMessage) {
    if msg.Channel == "mychannel" {
        // deliver message
    }
})
```

3. Publish inbound messages to the bus:

```go
bus.PublishInbound(&bus.InboundMessage{
    Channel:   "mychannel",
    ChatID:    chatID,
    SenderID:  senderID,
    Content:   content,
    TraceID:   traceID,
})
```

4. Add configuration fields to `gomikrobot/internal/config/config.go` under `ChannelsConfig`.

5. Initialize the channel in the gateway command (`gomikrobot/cmd/gomikrobot/cmd/gateway.go`).

### Adding a New CLI Command

1. Create a new file in `gomikrobot/cmd/gomikrobot/cmd/`.

2. Define a `cobra.Command` variable:

```go
var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "Description of my command",
    RunE: func(cmd *cobra.Command, args []string) error {
        // implementation
        return nil
    },
}
```

3. Register in `init()`:

```go
func init() {
    rootCmd.AddCommand(myCmd)
}
```

---

## 7. Runtime Settings

Runtime settings are stored in the `settings` table of the timeline SQLite database (`~/.gomikrobot/timeline.db`).

### Schema

```sql
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at DATETIME
);
```

### API Access

- **Read**: `GET /api/v1/settings`
- **Write**: `POST /api/v1/settings`

Programmatic access:

```go
value, err := timeline.GetSetting("key_name")
err := timeline.SetSetting("key_name", "value")
```

### Known Settings Keys

| Key | Type | Description |
|---|---|---|
| `whatsapp_allowlist` | Newline-separated JIDs | Approved WhatsApp senders |
| `whatsapp_denylist` | Newline-separated JIDs | Blocked WhatsApp senders |
| `whatsapp_pending` | Newline-separated JIDs | Senders awaiting approval |
| `whatsapp_pair_token` | String | Pairing token for WhatsApp authentication |
| `daily_token_limit` | Integer string | Daily token budget (see [Token Quota Management](#5-token-quota-management)) |
| `bot_repo_path` | Path string | Path to bot identity/system repository |
| `selected_repo_path` | Path string | Currently selected repository in the dashboard |
| `silent_mode` | `"true"` / `"false"` | When `"true"`, bot does not auto-respond. Defaults to `true` (safe default) if the setting is missing. |

---

## 8. Web User Management

### Web Users

Web users represent identities in the Web UI dashboard. They are stored in the `web_users` table.

**Schema:**

```sql
CREATE TABLE IF NOT EXISTS web_users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE,
    force_send BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**Operations:**

| Operation | Method | Description |
|---|---|---|
| Create user | `CreateWebUser(name)` | Creates a new user or returns existing by name |
| List users | `ListWebUsers()` | Returns all users sorted by name |
| Get user | `GetWebUser(id)` | Lookup by ID |
| Get user by name | `GetWebUserByName(name)` | Lookup by display name |
| Set force-send | `SetWebUserForceSend(id, bool)` | Toggle force delivery flag |

### Web Links (Cross-Channel Identity)

Web links associate a web user with a WhatsApp JID, enabling cross-channel identity.

**Schema:**

```sql
CREATE TABLE IF NOT EXISTS web_links (
    web_user_id INTEGER PRIMARY KEY,
    whatsapp_jid TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_web_links_whatsapp ON web_links(whatsapp_jid);
```

**Operations:**

| Operation | Method | Description |
|---|---|---|
| Link user | `LinkWebUser(webUserID, whatsappJID)` | Link web user to WhatsApp JID (upsert) |
| Unlink user | `UnlinkWebUser(webUserID)` | Remove the WhatsApp link |
| Get link | `GetWebLink(webUserID)` | Returns JID and whether a link exists |

**Dashboard API endpoints:**

- `GET /api/v1/weblinks` -- List web links
- `POST /api/v1/weblinks` -- Create or update a web link

### Force-Send

- `POST /api/v1/webusers/force` -- Forces delivery of a message to a specific user, bypassing silent mode.

### Web Chat

- `POST /api/v1/webchat/send` -- Send messages to the bot from the web UI. Messages go through the same agent loop as WhatsApp and CLI channels.

---

## 9. Audit and Compliance

GoMikroBot provides comprehensive audit logging through multiple mechanisms.

### Policy Decision Log

Every tool invocation triggers a policy evaluation that is logged regardless of the outcome.

**Schema:**

```sql
CREATE TABLE IF NOT EXISTS policy_decisions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trace_id TEXT,
    task_id TEXT,
    tool TEXT NOT NULL,
    tier INTEGER NOT NULL,
    sender TEXT,
    channel TEXT,
    allowed BOOLEAN NOT NULL,
    reason TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_policy_trace ON policy_decisions(trace_id);
CREATE INDEX IF NOT EXISTS idx_policy_task ON policy_decisions(task_id);
```

**Fields:**

| Field | Description |
|---|---|
| `trace_id` | End-to-end trace identifier linking all operations for a single request |
| `task_id` | The agent task that triggered the tool call |
| `tool` | Tool name (e.g., `exec`, `write_file`) |
| `tier` | Tool risk tier (0, 1, or 2) |
| `sender` | Sender identifier (phone number, user ID) |
| `channel` | Channel name (whatsapp, cli, web) |
| `allowed` | Whether the tool call was permitted |
| `reason` | Human-readable reason (e.g., `tier_0_always_allowed`, `sender_not_authorized`) |

**Query via dashboard:**

```
GET /api/v1/policy-decisions?trace_id=<trace_id>
```

### Task Tracking

Every inbound message creates an `AgentTask` record that tracks the full processing lifecycle.

**Task lifecycle:**

```
pending --> processing --> completed
                     \--> failed
```

**Schema (tasks table):**

```sql
CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id TEXT UNIQUE NOT NULL,
    idempotency_key TEXT UNIQUE,
    trace_id TEXT,
    channel TEXT NOT NULL,
    chat_id TEXT NOT NULL,
    sender_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    content_in TEXT,
    content_out TEXT,
    error_text TEXT,
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    delivery_status TEXT NOT NULL DEFAULT 'pending',
    delivery_attempts INTEGER NOT NULL DEFAULT 0,
    delivery_next_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);
```

**Audit-relevant fields:**

| Field | Description |
|---|---|
| `task_id` | Unique task identifier (128-bit random hex) |
| `idempotency_key` | Prevents duplicate processing of the same message |
| `trace_id` | Links task to timeline events and policy decisions |
| `content_in` | Full user input (preserved for audit) |
| `content_out` | Full bot response (preserved for audit) |
| `error_text` | Error details if task failed |
| `prompt_tokens` / `completion_tokens` / `total_tokens` | Token usage for cost attribution |
| `delivery_status` | Tracks whether response reached the user (`pending`, `sent`, `failed`, `skipped`) |

### Deduplication

The agent loop implements idempotency to prevent duplicate processing:

1. An `idempotency_key` is generated for each inbound message (format: `auto:<channel>:<trace_id>`).
2. Before processing, the system checks for an existing task with the same key.
3. If a completed task exists, the cached `content_out` is returned immediately.
4. If a task is still processing, the message is skipped.

### Timeline Event History

All interactions (messages, audio, images) are logged in the `timeline` table with full tracing support.

**Schema:**

```sql
CREATE TABLE IF NOT EXISTS timeline (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT UNIQUE,
    trace_id TEXT,
    span_id TEXT,
    parent_span_id TEXT,
    timestamp DATETIME,
    sender_id TEXT,
    sender_name TEXT,
    event_type TEXT,
    content_text TEXT,
    media_path TEXT,
    vector_id TEXT,
    classification TEXT,
    authorized BOOLEAN DEFAULT 1
);
```

**Event types:** `TEXT`, `AUDIO`, `IMAGE`, `SYSTEM`

**Tracing fields:**

| Field | Description |
|---|---|
| `trace_id` | End-to-end request trace |
| `span_id` | Individual operation span |
| `parent_span_id` | Parent span for nested operations |

The dashboard provides a trace viewer for drill-down exploration of individual request flows across timeline events, task records, and policy decisions.

### Querying Audit Data

**Timeline events with filters:**

```go
events, err := timeline.GetEvents(FilterArgs{
    SenderID:       "49123456789@s.whatsapp.net",
    TraceID:        "trace-1234567890",
    StartDate:      &startTime,
    EndDate:        &endTime,
    AuthorizedOnly: &trueVal,
    Limit:          100,
    Offset:         0,
})
```

**Policy decisions by trace:**

```go
decisions, err := timeline.ListPolicyDecisions("trace-1234567890")
```

**Tasks with filters:**

```go
tasks, err := timeline.ListTasks("completed", "whatsapp", 50, 0)
```

**Daily token usage:**

```go
total, err := timeline.GetDailyTokenUsage()
```

---

## Database Location

All persistent state (timeline events, settings, tasks, policy decisions, memory chunks, web users, web links) is stored in a single SQLite database:

```
~/.gomikrobot/timeline.db
```

The database uses WAL journal mode, foreign keys, and a 5-second busy timeout for concurrent access safety.
