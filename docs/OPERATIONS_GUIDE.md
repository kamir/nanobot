# GoMikroBot Operations Guide

This document covers the architecture, build pipeline, deployment, networking, database, observability, API surface, and operational procedures for GoMikroBot.

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Build & Release](#2-build--release)
3. [Deployment](#3-deployment)
4. [Network & Ports](#4-network--ports)
5. [Database](#5-database)
6. [Logging & Observability](#6-logging--observability)
7. [API Reference](#7-api-reference)
8. [Health Checks & Backup](#8-health-checks--backup)
9. [Graceful Shutdown](#9-graceful-shutdown)

---

## 1. Architecture Overview

### High-Level Data Flow

```
CLI/WhatsApp/Web --> Message Bus --> Agent Loop --> LLM Provider (OpenAI/OpenRouter)
                                        |
                                   Tool Registry --> Filesystem / Shell / Memory
                                        ^
                                   Context Builder (loads soul files from workspace/)
```

### Key Packages (`gomikrobot/internal/`)

| Package | Responsibility |
|---------|----------------|
| `agent/` | Core agent loop (`loop.go`) and context/soul-file loader (`context.go`). Orchestrates the entire request lifecycle. |
| `bus/` | Async message bus (pub-sub) with `InboundMessage` and `OutboundMessage` channels. Decouples input channels from the agent loop. |
| `channels/` | WhatsApp integration using `whatsmeow` (native Go, no Node bridge). Implements the `Channel` interface. |
| `config/` | Configuration loading with precedence: env vars > `~/.gomikrobot/config.json` > defaults. |
| `provider/` | LLM provider abstraction. Implementations for OpenAI, OpenRouter, local Whisper transcription, and TTS. |
| `session/` | Per-session conversation history with JSONL persistence. Thread-safe. |
| `timeline/` | SQLite event log (`~/.gomikrobot/timeline.db`) with trace/span IDs. Stores messages, settings, tasks, policy decisions, and web user mappings. |
| `tools/` | Registry-based tool system. Filesystem operations enforce path safety; shell execution uses deny-pattern filtering and configurable timeout (default 60s). |
| `memory/` | Semantic memory with vector embeddings. Supports SQLite-vec backend. Stores memory chunks with source and relevance metadata. |
| `policy/` | Policy engine for tool access control. Evaluates tier-based permissions per sender, channel, and tool. Logs all decisions. |

### Request Lifecycle

1. A message arrives via a channel (WhatsApp, CLI, or Web UI).
2. The channel publishes an `InboundMessage` to the message bus.
3. The agent loop consumes the message and creates a task record in the timeline database.
4. A deduplication check runs against the task's idempotency key to prevent reprocessing.
5. The context builder assembles the system prompt from soul files (`AGENTS.md`, `SOUL.md`, `USER.md`, `TOOLS.md`, `IDENTITY.md`) in the workspace directory.
6. RAG context is injected from semantic memory (top 5 results, relevance threshold >= 30%).
7. The LLM is called with tool definitions attached.
8. Tool calls are evaluated by the policy engine and executed if allowed.
9. The agentic loop iterates up to 20 times (`maxIterations`) until the LLM returns a final text response.
10. The response is published as an `OutboundMessage` on the bus.
11. The task status is updated in the timeline database (`completed` or `failed`).

### CLI Commands (`gomikrobot/cmd/gomikrobot/cmd/`)

| Command | Purpose |
|---------|---------|
| `gateway` | Main daemon: WhatsApp + CLI + Web UI on ports 18790/18791 |
| `agent -m "msg"` | Single-message mode for quick testing |
| `onboard` | First-time setup wizard |
| `status` | Version and configuration check |
| `whatsapp-setup` | WhatsApp configuration |
| `whatsapp-auth` | WhatsApp QR code authentication |
| `install` | System install to `/usr/local/bin` |

---

## 2. Build & Release

### Prerequisites

- Go 1.24.0+ (toolchain 1.24.13)
- All Go commands run from the `gomikrobot/` directory

### Make Targets

All targets are defined in `gomikrobot/Makefile`:

| Target | Description |
|--------|-------------|
| `make build` | Build the `gomikrobot` binary (`go build ./cmd/gomikrobot`) |
| `make run` | Build and run the gateway |
| `make rerun` | Kill existing processes on ports 18790/18791, rebuild, and run |
| `make install` | Install via the `gomikrobot install` command |
| `make release-patch` | Bump patch version via `scripts/release.sh` and build |
| `make release-minor` | Bump minor version and build |
| `make release-major` | Bump major version and build |
| `make docker-build` | Build the Go binary, then build Docker image `gomikrobot:local` |
| `make docker-up` | Start docker-compose with the local image |
| `make docker-down` | Stop docker-compose |
| `make docker-logs` | Tail docker-compose logs |
| `make help` | Show all available targets |

### Running Tests

```bash
cd gomikrobot

# All tests
go test ./...

# Single package
go test ./internal/tools/
go test ./internal/agent/
go test ./internal/memory/
```

### Versioning

Versioning is managed by `gomikrobot/scripts/release.sh` with semantic versioning (major/minor/patch). The script bumps the version, and the Make target triggers a build afterwards.

### CI/CD (GitHub Actions)

Workflow file: `.github/workflows/release-go.yml`

**Triggers:**
- Push of tags matching `v*`
- Manual `workflow_dispatch`

**Build matrix:**
- `ubuntu-latest`
- `macos-latest`
- `windows-latest`

**Steps:**
1. Checkout repository
2. Setup Go 1.24.13
3. Build binary to `dist/` directory (with `.exe` suffix on Windows)
4. Upload build artifact per OS

**Release creation:**
- Runs only on tag pushes (`refs/tags/v*`)
- Downloads all platform artifacts
- Creates a GitHub Release using `softprops/action-gh-release@v2`
- Attaches all platform binaries

---

## 3. Deployment

### Local Deployment

```bash
cd gomikrobot
make build
./gomikrobot onboard   # first-time setup
./gomikrobot gateway   # start the daemon
```

The `onboard` command performs first-time configuration. After that, `gateway` starts all services (WhatsApp channel, agent loop, API server, dashboard).

### Docker Deployment

**Dockerfile** (`gomikrobot/Dockerfile`):
- Base image: `alpine:3.20`
- Installs `ca-certificates`
- Copies pre-built `gomikrobot` binary to `/usr/local/bin/gomikrobot`
- Copies `web/` directory to `/app/web`
- Exposes ports `18790` and `18791`
- Entrypoint: `/usr/local/bin/gomikrobot`
- Default command: `gateway`

**docker-compose.yml** (`gomikrobot/docker-compose.yml`):

```yaml
services:
  gomikrobot:
    image: gomikrobot:local
    pull_policy: never
    container_name: gomikrobot
    ports:
      - "18790:18790"
      - "18791:18791"
    environment:
      MIKROBOT_AGENTS_SYSTEM_REPO_PATH: /opt/system-repo
      MIKROBOT_AGENTS_WORK_REPO_PATH: /opt/work-repo
      MIKROBOT_AGENTS_WORKSPACE: /root/.gomikrobot/workspace
    volumes:
      - ${SYSTEM_REPO_PATH}:/opt/system-repo
      - ${WORK_REPO_PATH}:/opt/work-repo
      - ${HOME}/.gomikrobot:/root/.gomikrobot
```

**Required host environment variables for docker-compose:**

| Variable | Description |
|----------|-------------|
| `SYSTEM_REPO_PATH` | Path to the system/identity repo on the host |
| `WORK_REPO_PATH` | Path to the work repo on the host (default: `~/GoMikroBot-Workspace`) |

**Commands:**

```bash
# Build binary + Docker image
make docker-build

# Start (builds first, then runs detached)
make docker-up

# Stop
make docker-down

# View logs
make docker-logs
```

Note: `make docker-up` defaults `WORK_REPO_PATH` to `$HOME/GoMikroBot-Workspace` if not set. `SYSTEM_REPO_PATH` defaults to the current directory.

### System Install

```bash
# Install binary to /usr/local/bin
./gomikrobot install

# Or with sudo if needed
sudo ./gomikrobot install
```

---

## 4. Network & Ports

| Port | Service | Protocol | Description |
|------|---------|----------|-------------|
| 18790 | API Server | HTTP | Message submission endpoint (`POST /chat`). Used for programmatic access and local network integration. |
| 18791 | Dashboard / Web UI | HTTP | SPA dashboard serving the timeline view and all REST API endpoints. Auto-refreshes every 5 seconds. |

**Default bind address:** `127.0.0.1` (localhost only, secure default).

To change the bind address or ports, update `~/.gomikrobot/config.json`:

```json
{
  "gateway": {
    "host": "127.0.0.1",
    "port": 18790,
    "dashboardPort": 18791
  }
}
```

Or use environment variables:
- `MIKROBOT_HOST`
- `MIKROBOT_PORT`
- `MIKROBOT_DASHBOARD_PORT`

**CORS:** All dashboard API endpoints include `Access-Control-Allow-Origin: *` headers.

---

## 5. Database

### Location

SQLite database at `~/.gomikrobot/timeline.db`. Uses WAL (Write-Ahead Logging) mode for concurrent read access.

### Schema

The database contains seven tables:

#### `timeline`

The main event log. Stores every inbound/outbound message and system event.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER | Auto-incrementing primary key |
| `event_id` | TEXT | Unique event identifier (e.g., WhatsApp MessageID) |
| `trace_id` | TEXT | End-to-end trace identifier |
| `span_id` | TEXT | Span identifier (optional) |
| `parent_span_id` | TEXT | Parent span (optional) |
| `timestamp` | DATETIME | When the event occurred |
| `sender_id` | TEXT | Sender identifier (phone number, session key) |
| `sender_name` | TEXT | Display name |
| `event_type` | TEXT | `TEXT`, `AUDIO`, `IMAGE`, `SYSTEM` |
| `content_text` | TEXT | The message text or transcript |
| `media_path` | TEXT | Path to local media file |
| `vector_id` | TEXT | Qdrant vector ID (if applicable) |
| `classification` | TEXT | Event classification (e.g., `LOCAL_INBOUND`, `WEBUI_OUTBOUND`) |
| `authorized` | BOOLEAN | Whether the sender is in the allow list |

Indexes: `timestamp`, `sender_id`, `authorized`

#### `settings`

Key-value store for runtime settings.

| Column | Type | Description |
|--------|------|-------------|
| `key` | TEXT | Setting name (primary key) |
| `value` | TEXT | Setting value |
| `updated_at` | DATETIME | Last update timestamp |

Known settings keys:
- `whatsapp_allowlist` -- Comma-separated allowed WhatsApp JIDs
- `whatsapp_denylist` -- Comma-separated denied WhatsApp JIDs
- `whatsapp_pending` -- Pending WhatsApp authorization requests
- `whatsapp_pair_token` -- WhatsApp pairing token
- `daily_token_limit` -- Maximum daily LLM token usage
- `bot_repo_path` -- System/identity repository path
- `work_repo_path` -- Active work repository path
- `selected_repo_path` -- Currently selected repo in the UI
- `silent_mode` -- When enabled, suppresses outbound WhatsApp delivery
- `default_work_repo_path` -- Default work repo path
- `default_repo_search_path` -- Root path for repo discovery
- `kafscale_lfs_proxy_url` -- LFS proxy URL

#### `web_users`

Web UI user identities.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER | Auto-incrementing primary key |
| `name` | TEXT | Unique user name |
| `force_send` | BOOLEAN | Override silent mode for this user (default: true) |
| `created_at` | DATETIME | Creation timestamp |

#### `web_links`

Maps web users to WhatsApp JIDs.

| Column | Type | Description |
|--------|------|-------------|
| `web_user_id` | INTEGER | Primary key, references `web_users.id` |
| `whatsapp_jid` | TEXT | WhatsApp JID |
| `created_at` | DATETIME | Creation timestamp |
| `updated_at` | DATETIME | Last update timestamp |

Index: `whatsapp_jid`

#### `tasks`

Agent task tracking with full lifecycle management.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER | Auto-incrementing primary key |
| `task_id` | TEXT | Unique task identifier |
| `idempotency_key` | TEXT | Deduplication key (unique) |
| `trace_id` | TEXT | Trace identifier |
| `channel` | TEXT | Source channel (`whatsapp`, `cli`, `webui`) |
| `chat_id` | TEXT | Chat/conversation identifier |
| `sender_id` | TEXT | Sender identifier |
| `status` | TEXT | `pending`, `processing`, `completed`, `failed` |
| `content_in` | TEXT | Input message content |
| `content_out` | TEXT | Agent response content |
| `error_text` | TEXT | Error message if failed |
| `prompt_tokens` | INTEGER | LLM prompt token count |
| `completion_tokens` | INTEGER | LLM completion token count |
| `total_tokens` | INTEGER | Total token count |
| `delivery_status` | TEXT | `pending`, `sent`, `failed`, `skipped` |
| `delivery_attempts` | INTEGER | Number of delivery retries |
| `delivery_next_at` | DATETIME | Next retry time (exponential backoff) |
| `created_at` | DATETIME | Task creation time |
| `updated_at` | DATETIME | Last update time |
| `completed_at` | DATETIME | Completion time |

Indexes: `status`, `idempotency_key`, `trace_id`, `(delivery_status, delivery_next_at)`

#### `policy_decisions`

Audit log for all tool access policy evaluations.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER | Auto-incrementing primary key |
| `trace_id` | TEXT | Trace identifier |
| `task_id` | TEXT | Associated task identifier |
| `tool` | TEXT | Tool name |
| `tier` | INTEGER | Tool tier (security level) |
| `sender` | TEXT | Sender identifier |
| `channel` | TEXT | Source channel |
| `allowed` | BOOLEAN | Whether the tool call was permitted |
| `reason` | TEXT | Reason for the decision |
| `created_at` | DATETIME | Decision timestamp |

Indexes: `trace_id`, `task_id`

#### `memory_chunks`

Semantic memory storage for RAG context injection.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | Primary key |
| `content` | TEXT | Memory content |
| `embedding` | BLOB | Vector embedding |
| `source` | TEXT | Source label (default: `user`) |
| `tags` | TEXT | Comma-separated tags |
| `version` | INTEGER | Schema version |
| `created_at` | DATETIME | Creation timestamp |
| `updated_at` | DATETIME | Last update timestamp |

Index: `source`

---

## 6. Logging & Observability

### Structured Logging

GoMikroBot uses Go's standard `log/slog` package for structured logging. Log output includes key-value pairs for context:

```
INFO Agent loop started
INFO Delivery worker started interval=5s max_retry=5
DEBUG Tool executed name=read_file result_length=1234
WARN RAG search failed error=...
ERROR Failed to process message error=...
```

### Tracing

Every message gets a trace ID on ingestion:
- Format: `trace-{unix_nano}` (agent loop) or hex-encoded random bytes (gateway HTTP endpoints)
- Trace IDs link all events, tasks, and policy decisions for a single request
- Retrievable via `GET /api/v1/trace/{traceID}`

### Token Usage Tracking

- Token usage (prompt, completion, total) is tracked per task in the `tasks` table
- Daily token usage aggregation is available via the timeline service
- A configurable `daily_token_limit` setting enforces quota checks before each LLM call
- When the quota is exceeded, the agent returns an error message rather than making the LLM call

### Policy Audit Trail

Every tool call evaluation is logged to the `policy_decisions` table with:
- Trace ID and task ID for correlation
- Tool name and tier
- Sender and channel information
- Allow/deny decision with reason

Retrieve policy decisions via: `GET /api/v1/policy-decisions?trace_id={traceID}`

### Task Lifecycle

Tasks follow this state machine:

```
pending --> processing --> completed
                      \-> failed
```

Delivery states:

```
pending --> sent --> delivered
       \-> failed
       \-> skipped
```

The delivery worker polls every 5 seconds for tasks with pending delivery. It retries up to 5 times with exponential backoff (30s * 2^attempts, max 5 minutes).

---

## 7. API Reference

### Port 18790 -- API Server

#### `POST /chat`

Send a message to the agent for processing.

**Query parameters:**
- `message` (required) -- The message text
- `session` (optional) -- Session key (default: `local:default`)

**Response:** Plain text agent response.

**Example:**

```bash
curl -X POST "http://127.0.0.1:18790/chat?message=hello&session=local:default"
```

---

### Port 18791 -- Dashboard API

#### Navigation

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | Redirects to `/timeline` |
| `GET` | `/timeline` | SPA dashboard (serves `web/timeline.html`) |
| `GET` | `/media/{path}` | Static media file server |

#### Timeline

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/timeline` | Get timeline events |

**Query parameters:** `limit` (default: 100), `offset`, `sender`, `trace_id`

#### Traces

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/trace/{traceID}` | Get trace detail with spans, task info, and policy decisions |
| `GET` | `/api/v1/policy-decisions?trace_id={traceID}` | Get policy decisions for a specific trace |

#### Settings

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/settings` | Get settings. Use `?key=name` for a specific key. Default returns `silent_mode` status. |
| `POST` | `/api/v1/settings` | Update a setting. Body: `{"key": "...", "value": "..."}` |

#### Work Repo

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/workrepo` | Get current work repo path |
| `POST` | `/api/v1/workrepo` | Set work repo path. Body: `{"path": "/absolute/path"}` |

#### Repository Operations

All repository endpoints operate on the current work repo by default. Add `?repo=identity` to target the system/identity repo instead.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/repo/tree` | Browse directory tree. Optional `?path=subdir` |
| `GET` | `/api/v1/repo/file` | Read file contents. Required `?path=relative/path` |
| `GET` | `/api/v1/repo/status` | Git status (`-sb` format) and remote info |
| `GET` | `/api/v1/repo/search` | Search for git repositories under the configured search path |
| `GET` | `/api/v1/repo/gh-auth` | Check GitHub CLI authentication status |
| `GET` | `/api/v1/repo/branches` | List local branches |
| `POST` | `/api/v1/repo/checkout` | Checkout a branch. Body: `{"branch": "name"}` |
| `GET` | `/api/v1/repo/log` | Commit log. Optional `?limit=20` |
| `GET` | `/api/v1/repo/diff-file` | Diff a single file. Required `?path=relative/path` |
| `GET` | `/api/v1/repo/diff` | Full repo diff. Optional `?path=relative/path` |
| `POST` | `/api/v1/repo/commit` | Stage all and commit. Body: `{"message": "..."}` |
| `POST` | `/api/v1/repo/pull` | Pull from remote (fast-forward only) |
| `POST` | `/api/v1/repo/push` | Push to remote |
| `POST` | `/api/v1/repo/init` | Initialize git repo. Optional body: `{"remote_url": "..."}` |
| `POST` | `/api/v1/repo/pr` | Create a pull request via GitHub CLI. Body: `{"title": "...", "body": "...", "base": "...", "head": "...", "draft": false}` |

#### Web Users

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/webusers` | List all web users |
| `POST` | `/api/v1/webusers` | Create a web user. Body: `{"name": "..."}` |
| `POST` | `/api/v1/webusers/force` | Set force-send flag. Body: `{"web_user_id": 1, "force_send": true}` |

#### Web Links

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/weblinks?web_user_id=1` | Get WhatsApp link for a web user |
| `POST` | `/api/v1/weblinks` | Create or remove link. Body: `{"web_user_id": 1, "whatsapp_jid": "..."}` (empty JID removes link) |

#### Web Chat

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/webchat/send` | Send a message from the web UI. Body: `{"web_user_id": 1, "message": "..."}` |

#### Tasks

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/tasks` | List tasks. Optional: `?status=completed&channel=whatsapp&limit=50&offset=0` |
| `GET` | `/api/v1/tasks/{taskID}` | Get task details by task ID |

---

## 8. Health Checks & Backup

### Health Checks

There is no dedicated `/health` endpoint at this time. To verify the gateway is running:

```bash
# Check API server (port 18790)
curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:18790/chat

# Check dashboard (port 18791)
curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:18791/timeline

# Check that both ports are listening
lsof -i tcp:18790 -sTCP:LISTEN
lsof -i tcp:18791 -sTCP:LISTEN
```

### Backup

The following files and directories should be backed up regularly:

| Path | Description |
|------|-------------|
| `~/.gomikrobot/timeline.db` | Main database (timeline, tasks, settings, policy decisions, memory) |
| `~/.gomikrobot/whatsapp.db` | WhatsApp session database (authentication state) |
| `~/.gomikrobot/config.json` | Configuration file |
| `~/.gomikrobot/workspace/` | Workspace directory (soul files, session logs, media) |

**Backup procedure:**

```bash
# Stop the gateway first for a consistent snapshot, or copy while running
# (SQLite WAL mode supports concurrent reads)
BACKUP_DIR="$HOME/gomikrobot-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"
cp ~/.gomikrobot/timeline.db "$BACKUP_DIR/"
cp ~/.gomikrobot/whatsapp.db "$BACKUP_DIR/" 2>/dev/null || true
cp ~/.gomikrobot/config.json "$BACKUP_DIR/" 2>/dev/null || true
cp -r ~/.gomikrobot/workspace "$BACKUP_DIR/" 2>/dev/null || true
```

For a consistent SQLite backup without stopping the gateway, use the SQLite backup API:

```bash
sqlite3 ~/.gomikrobot/timeline.db ".backup '$BACKUP_DIR/timeline.db'"
```

---

## 9. Graceful Shutdown

The gateway uses Go's `context.Context` cancellation pattern for coordinated shutdown.

### Signal Handling

The gateway listens for `SIGINT` (Ctrl+C) and `SIGTERM` signals:

```
Signal received
    |
    v
WhatsApp channel stopped (wa.Stop())
    |
    v
Agent loop stopped (loop.Stop())
    |
    v
Timeline database closed (timeSvc.Close())
    |
    v
Process exits
```

### Shutdown Sequence

1. A `SIGINT` or `SIGTERM` signal is received.
2. The context is cancelled, propagating to all goroutines.
3. The WhatsApp channel disconnects cleanly.
4. The agent loop checks `ctx.Err()` on each iteration and exits when the context is cancelled.
5. The delivery worker stops polling.
6. The bus dispatcher stops.
7. The timeline database connection is closed.
8. The process exits.

### Port Cleanup

If the gateway did not shut down cleanly (e.g., after a crash), use `make rerun` to automatically kill any leftover processes on ports 18790 and 18791 before restarting:

```bash
make rerun
```

Or manually:

```bash
# Find and kill processes on the gateway ports
lsof -ti tcp:18790 -sTCP:LISTEN | xargs kill
lsof -ti tcp:18791 -sTCP:LISTEN | xargs kill
```

### Dashboard Failure

If the dashboard server fails to bind its port, it triggers a context cancellation that stops the entire gateway. This is by design -- the dashboard is considered essential for operation.
