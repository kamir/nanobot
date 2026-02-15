# GoMikroBot User Manual

A comprehensive guide to installing, configuring, and using GoMikroBot -- a lightweight, ultra-fast personal AI assistant framework written in Go.

---

## Table of Contents

1. [Getting Started](#1-getting-started)
2. [Quick Start](#2-quick-start)
3. [CLI Reference](#3-cli-reference)
4. [Web Dashboard](#4-web-dashboard)
5. [WhatsApp Integration](#5-whatsapp-integration)
6. [Memory System](#6-memory-system)
7. [Day2Day Task Tracker](#7-day2day-task-tracker)
8. [Soul Files and Workspace](#8-soul-files-and-workspace)
9. [FAQ / Troubleshooting](#9-faq--troubleshooting)

---

## 1. Getting Started

### Prerequisites

- **Go 1.24.0+** (toolchain 1.24.13)
- **OpenAI API key** (or an OpenRouter API key)
- **Operating System:** macOS / Linux / Windows

### Installation

Clone the repository and build from source:

```bash
cd gomikrobot
go build ./cmd/gomikrobot
```

Or use the Makefile:

```bash
cd gomikrobot
make build
```

To install the binary system-wide to `/usr/local/bin`:

```bash
./gomikrobot install
# May require sudo depending on your system permissions
```

Or use the Makefile targets for building and running in one step:

```bash
make run       # build + run the gateway
make rerun     # kill existing processes on ports 18790/18791, rebuild, run
make install   # (if available in your Makefile)
```

### First-Time Setup

Run the onboard command to create the default configuration:

```bash
./gomikrobot onboard
```

This creates `~/.gomikrobot/config.json` with sensible defaults. After onboarding, edit the config file to add your API keys:

```bash
# Edit config to add your API key
# Set OPENAI_API_KEY as an env var, or edit ~/.gomikrobot/config.json directly
export OPENAI_API_KEY="sk-..."
```

Verify everything is working:

```bash
./gomikrobot status
```

---

## 2. Quick Start

After completing the setup above, the fastest way to start using GoMikroBot:

```bash
# 1. Build
cd gomikrobot
go build ./cmd/gomikrobot

# 2. Initialize configuration
./gomikrobot onboard          # creates ~/.gomikrobot/config.json

# 3. Add your API key (edit config.json or set env var)
export OPENAI_API_KEY="sk-..."

# 4. Test with a single message
./gomikrobot agent -m "hello"

# 5. Start the full gateway (WhatsApp + CLI + Web UI)
./gomikrobot gateway
```

Once the gateway is running:

- **API server** listens on `http://localhost:18790`
- **Web dashboard** is available at `http://localhost:18791`
- **WhatsApp** channel connects automatically if configured

---

## 3. CLI Reference

GoMikroBot provides 8 CLI commands. Run `gomikrobot` with no arguments (or `gomikrobot --help`) to see the full list.

### 3.1 `gateway`

Start the agent gateway daemon. This is the main long-running process that hosts the WhatsApp channel, CLI input, Web UI, and API server.

```
Usage: gomikrobot gateway
```

- **Flags:** None
- **Ports:**
  - `18790` -- API server (accepts `/chat` POST requests)
  - `18791` -- Web dashboard
- **Behavior:** Runs until interrupted with Ctrl+C. Handles graceful shutdown of WhatsApp, the agent loop, and the timeline database.

```bash
gomikrobot gateway
```

### 3.2 `agent`

Chat with the agent directly from the command line in single-message mode.

```
Usage: gomikrobot agent [flags]
```

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--message` | `-m` | string | *(required)* | Message to send to the agent |
| `--session` | `-s` | string | `cli:default` | Session ID for conversation continuity |

```bash
# Basic usage
gomikrobot agent -m "hello"

# With a specific session
gomikrobot agent -m "what did we discuss?" -s "cli:project-x"
```

### 3.3 `onboard`

Initialize GoMikroBot configuration. Creates `~/.gomikrobot/config.json` with default settings and ensures the workspace directory exists.

```
Usage: gomikrobot onboard [flags]
```

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--force` | `-f` | bool | `false` | Overwrite existing config.json |

```bash
# First-time setup
gomikrobot onboard

# Reset configuration to defaults
gomikrobot onboard --force
```

After onboarding, you will see next-step instructions:

1. Edit `config.json` to add your API keys.
2. Run `gomikrobot agent -m "hello"` to test.

### 3.4 `status`

Show system status including version, configuration state, API key presence, and WhatsApp connectivity.

```
Usage: gomikrobot status
```

- **Flags:** None
- **Output includes:**
  - Version number
  - Config file presence (`~/.gomikrobot/config.json`)
  - API key availability
  - WhatsApp enabled/disabled status
  - WhatsApp session status (linked or QR needed)
  - QR code file path (`~/.gomikrobot/whatsapp-qr.png`)

```bash
gomikrobot status
```

### 3.5 `version`

Print the current version of GoMikroBot.

```
Usage: gomikrobot version
```

- **Flags:** None
- **Current version:** 1.1.0 (can be overridden at build time via `-ldflags`)

```bash
gomikrobot version
```

### 3.6 `install`

Install the `gomikrobot` binary to `/usr/local/bin` for system-wide access.

```
Usage: gomikrobot install
```

- **Flags:** None
- **Note:** May require `sudo` privileges depending on file system permissions. If a `scripts/install.sh` script exists alongside the binary, it will be used instead of a direct copy.

```bash
gomikrobot install
# or, if permission denied:
sudo ./gomikrobot install
```

### 3.7 `whatsapp-setup`

Interactive wizard to configure WhatsApp authentication, including the pairing token and allow/deny lists.

```
Usage: gomikrobot whatsapp-setup
```

- **Flags:** None
- **Prompts for:**
  1. Enable WhatsApp channel (y/N)
  2. Pairing token (shared out-of-band for device linking)
  3. Initial allowlist (comma or newline separated JIDs, optional)
  4. Initial denylist (comma or newline separated JIDs, optional)

Settings are stored in both `~/.gomikrobot/config.json` (enabled flag) and the timeline SQLite database (token, allowlist, denylist).

```bash
gomikrobot whatsapp-setup
```

### 3.8 `whatsapp-auth`

Manage WhatsApp JID authorization. Approve or deny JIDs from the pending list, or list all current allow/deny/pending JIDs.

```
Usage: gomikrobot whatsapp-auth [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--approve` | string | `""` | Approve a pending JID (move to allowlist) |
| `--deny` | string | `""` | Deny a pending JID (move to denylist) |
| `--list` | bool | `false` | List all allow/deny/pending JID lists |

```bash
# View all JID lists
gomikrobot whatsapp-auth --list

# Approve a specific JID
gomikrobot whatsapp-auth --approve "+1234567890@s.whatsapp.net"

# Deny a specific JID
gomikrobot whatsapp-auth --deny "+0987654321@s.whatsapp.net"
```

---

## 4. Web Dashboard

The web dashboard provides a browser-based interface for monitoring and interacting with GoMikroBot.

### Access

When the gateway is running, open your browser to:

```
http://localhost:18791
```

The root URL (`/`) redirects to `/timeline`.

### Features

- **Timeline View** (`/timeline`) -- Shows the full conversation history with trace IDs, sender information, event types, and classification labels. Auto-refreshes every 5 seconds.

- **Trace Viewer** -- Drill into individual trace spans for a specific request. Shows inbound, outbound, LLM, and tool execution spans. Also displays task metadata (token counts, delivery status) and policy decisions for that trace.

- **Task List** (`/api/v1/tasks`) -- View agent tasks with status (pending, processing, completed, failed), channel, token usage, and delivery status. Supports filtering by status and channel.

- **Repository Browser** -- Browse files in the work repo or system repo. View file contents, diffs, commit history, and branch information. Supports:
  - File tree navigation
  - File content viewing (text files up to 200KB)
  - Git diff viewing (per-file and full repo)
  - Commit with message
  - Pull (fast-forward only)
  - Push
  - Branch listing and checkout
  - Repository initialization with remote URL
  - Pull request creation via `gh`

- **Web Chat** -- Send messages to the bot directly from the browser. Messages are processed by the agent loop and responses are delivered through the configured channels. Supports web user management and WhatsApp JID linking.

- **Settings Panel** (`/api/v1/settings`) -- Configure runtime settings including silent mode. Settings are stored in the timeline SQLite database and take effect immediately.

- **Floating/Dockable Panels** -- The dashboard UI uses floating panels that can be repositioned.

### API Endpoints

The dashboard server exposes the following REST API at port 18791:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/timeline` | GET | Query timeline events (limit, offset, sender, trace_id) |
| `/api/v1/trace/{traceID}` | GET | Get spans, task info, and policy decisions for a trace |
| `/api/v1/tasks` | GET | List agent tasks (status, channel, limit, offset) |
| `/api/v1/tasks/{taskID}` | GET | Get a specific task by ID |
| `/api/v1/settings` | GET/POST | Read or update runtime settings |
| `/api/v1/workrepo` | GET/POST | Get or change the active work repo path |
| `/api/v1/repo/tree` | GET | Browse files in the repo (path, repo=identity) |
| `/api/v1/repo/file` | GET | Read a file from the repo |
| `/api/v1/repo/status` | GET | Git status of the repo |
| `/api/v1/repo/diff` | GET | Git diff (optional path filter) |
| `/api/v1/repo/diff-file` | GET | Git diff for a specific file |
| `/api/v1/repo/log` | GET | Git log (limit, default 20) |
| `/api/v1/repo/branches` | GET | List branches |
| `/api/v1/repo/checkout` | POST | Checkout a branch |
| `/api/v1/repo/commit` | POST | Stage all + commit with message |
| `/api/v1/repo/pull` | POST | Git pull (fast-forward only) |
| `/api/v1/repo/push` | POST | Git push |
| `/api/v1/repo/init` | POST | Initialize repo, optionally set remote |
| `/api/v1/repo/search` | GET | Search for git repos under the configured search path |
| `/api/v1/repo/gh-auth` | GET | Check GitHub CLI auth status |
| `/api/v1/repo/pr` | POST | Create a pull request via `gh` |
| `/api/v1/webusers` | GET/POST | List or create web users |
| `/api/v1/webusers/force` | POST | Toggle force-send for a web user |
| `/api/v1/weblinks` | GET/POST | Get or set web user to WhatsApp JID links |
| `/api/v1/webchat/send` | POST | Send a message as a web user |
| `/api/v1/policy-decisions` | GET | Get policy decisions for a trace |

The API server on port 18790 exposes:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/chat` | POST | Send a message (query params: `message`, `session`) |

---

## 5. WhatsApp Integration

GoMikroBot uses the `whatsmeow` library for native Go WhatsApp connectivity. There is no Node.js bridge required.

### Setup Flow

1. **Configure WhatsApp:**
   ```bash
   gomikrobot whatsapp-setup
   ```
   Follow the interactive prompts to enable WhatsApp, set a pairing token, and define initial allowlists/denylists.

2. **Start the gateway:**
   ```bash
   gomikrobot gateway
   ```

3. **Link your device:**
   On first run, a QR code is generated and saved to `~/.gomikrobot/whatsapp-qr.png`. Open this image and scan it with WhatsApp on your phone (Settings > Linked Devices > Link a Device).

4. **Session persistence:**
   Once linked, the WhatsApp session is stored at `~/.gomikrobot/whatsapp.db`. Subsequent gateway starts will reconnect automatically without needing a new QR scan.

### Authorization Model

GoMikroBot uses a three-tier JID authorization system:

- **Allowlist** -- JIDs that are authorized to interact with the bot. Messages are processed normally.
- **Denylist** -- JIDs that are explicitly blocked. Messages are silently dropped.
- **Pending** -- Unknown senders are placed in the pending list. Their messages are held until an administrator approves or denies them.

Manage JID authorization with:

```bash
# List all JIDs
gomikrobot whatsapp-auth --list

# Approve a pending sender
gomikrobot whatsapp-auth --approve "+1234567890@s.whatsapp.net"

# Deny a pending sender
gomikrobot whatsapp-auth --deny "+1234567890@s.whatsapp.net"
```

### Silent Mode

Silent mode can be enabled via the Settings panel in the web dashboard or the `/api/v1/settings` API. When enabled:

- Outbound WhatsApp messages are suppressed (logged as `suppressed` in the timeline)
- **Force-send override:** Individual web users can have force-send enabled, which bypasses silent mode for their linked WhatsApp JID

### Key Files

| File | Purpose |
|------|---------|
| `~/.gomikrobot/config.json` | WhatsApp enabled flag |
| `~/.gomikrobot/whatsapp.db` | Session/device link persistence |
| `~/.gomikrobot/whatsapp-qr.png` | QR code for initial device linking |
| `~/.gomikrobot/timeline.db` | Stores allowlist, denylist, pending lists, and pairing token |

---

## 6. Memory System

GoMikroBot includes a semantic memory system powered by vector embeddings.

### Overview

The memory system is automatically initialized when the LLM provider supports embeddings (e.g., OpenAI). On startup, the gateway logs whether memory is active:

```
Memory system initialized
```

or:

```
Memory system disabled (provider does not support embeddings)
```

### Tools

Two tools are available to the agent for memory operations:

#### `remember`

Store a piece of information in long-term memory for later recall.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `content` | string | Yes | The information to remember |
| `tags` | string | No | Comma-separated tags for categorization |

The content is stored with a `user` source label and embedded for semantic search.

#### `recall`

Search long-term memory for information relevant to a query.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | The search query to find relevant memories |
| `limit` | integer | No | Maximum number of results (default: 5) |

Returns matching memories sorted by relevance score.

### RAG Context Injection

On every user message, GoMikroBot automatically searches semantic memory for relevant context:

- **Top 5 results** are retrieved
- Results with a **relevance score >= 30%** are injected into the system prompt
- This provides the agent with relevant background information without the user needing to explicitly recall it

### Memory File Storage

In addition to the vector store, memory is persisted as markdown files in the work repo:

| Path | Description |
|------|-------------|
| `{work-repo}/memory/MEMORY.md` | Main memory file, loaded into system prompt |
| `{work-repo}/memory/YYYY-MM-DD.md` | Daily memory notes |

### Soul File Indexing

On gateway startup, soul files (see [Section 8](#8-soul-files-and-workspace)) are automatically indexed into the memory system in the background. This ensures the agent's identity and behavioral context are available for semantic retrieval.

---

## 7. Day2Day Task Tracker

The Day2Day task tracker is a built-in daily task management system. Commands are sent as messages to the bot (via any channel: CLI, WhatsApp, or Web UI).

### Commands

| Command | Description |
|---------|-------------|
| `dtu [text]` | **Update task.** If text is provided, adds it as a new task immediately. If no text is provided, enters capture mode. |
| `dtp [text]` | **Progress update.** If text is provided, logs it as a progress entry. If no text is provided, enters capture mode. |
| `dts` | **Summarize.** Consolidate and summarize today's tasks. De-duplicates tasks, counts open/done, and updates the consolidated state. |
| `dtn` | **Plan next.** Suggest the next task to work on (first open task). |
| `dta` | **Plan all.** List all open tasks as a prioritized plan. |
| `dtc` | **Close capture.** Submit the buffered content from capture mode and end the capture session. |

### Capture Mode

Capture mode allows multi-line input for task updates or progress entries:

1. Send `dtu` or `dtp` with no text to start capture mode.
2. Send content lines -- each message is appended to the capture buffer.
3. Send `dtc` to close capture mode and submit all buffered content.

```
User:  dtu
Bot:   Day2Day: dtu capture started. Send dtc to close.
User:  Fix the login page CSS
User:  Update the API rate limiter
User:  dtc
Bot:   Aktualisiert. Naechster Schritt: Fix the login page CSS
```

### Task Status Query

You can ask for the status of tasks on a specific date by sending a natural language query containing the words "status" and "task" (or "day2day" / "aufgabe"):

```
User:  status task today
User:  status task 2026-02-14
User:  status task yesterday
```

The bot responds with open/done counts, the next suggested step, and a list of open tasks.

### Task File Format

Day2Day tasks are stored as markdown files in the system repo at:

```
{system-repo}/operations/day2day/tasks/YYYY-MM-DD.md
```

Each file has the following sections:

```markdown
# Day2Day -- 2026-02-14 (Friday)

## Tasks
- [ ] Open task one
- [ ] Open task two
- [x] Completed task

## Progress Log
- 14:30: UPDATE -- Added new tasks
- 15:00: PROGRESS -- Working on task one

## Notes / Context

## Consolidated State
- Open: 2
- Done: 1
- Last Consolidation: 15:30

## Next Step
- Open task one
```

---

## 8. Soul Files and Workspace

### Workspace Structure

The default workspace directory is `~/.gomikrobot/workspace/` (configurable in `config.json`). This is where the agent's personality, knowledge, and behavior are defined.

### Bootstrap Files

The following files are loaded automatically from the workspace directory at startup and assembled into the system prompt:

| File | Purpose |
|------|---------|
| `AGENTS.md` | Agent governance rules and operational constraints |
| `SOUL.md` | Core personality and behavioral guidelines |
| `USER.md` | User-specific preferences and context |
| `TOOLS.md` | Tool usage guidelines and restrictions |
| `IDENTITY.md` | Agent identity and naming |

These files are concatenated into the system prompt in the order listed above. If a file does not exist, it is silently skipped.

### Work Repo

The work repo is the agent's exclusive write target. All file operations (write, edit) are restricted to this directory by default.

- **Default path:** `~/.gomikrobot/work-repo/` (or `~/GoMikroBot-Workspace/`)
- **Configurable via:** `config.json` (`agents.defaults.workRepoPath`), runtime settings (`work_repo_path`), or the web dashboard
- **Artifact directories within the work repo:**
  - `memory/` -- Memory files (`MEMORY.md`, daily notes)
  - `requirements/` -- Behavior specifications
  - `tasks/` -- Plans and milestones
  - `docs/` -- Explanations and summaries

### Skills Directory

Custom skills can be defined in the workspace:

```
{workspace}/skills/{skill-name}/SKILL.md
```

Each skill is a markdown file describing a specific capability or behavioral pattern the agent should follow.

### System Repo Skills

The system repo (the GoMikroBot source code repository) can also provide skills and guidance:

- `{system-repo}/skills/{skill-name}/SKILL.md` -- Skill definitions
- `{system-repo}/operations/day2day/README.md` -- Day2Day operational guidance

These are loaded automatically into the system prompt alongside workspace skills.

### Configuration Hierarchy

Configuration is loaded in the following order (later values override earlier ones):

1. **Default values** (hardcoded in `DefaultConfig()`)
2. **Config file** (`~/.gomikrobot/config.json`)
3. **Environment variables** (prefix: `MIKROBOT_`)
4. **Runtime settings** (stored in `~/.gomikrobot/timeline.db`, modifiable via web dashboard)

Key environment variables:

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | OpenAI API key |
| `OPENROUTER_API_KEY` | OpenRouter API key |
| `MIKROBOT_OPENAI_API_KEY` | OpenAI API key (prefixed) |
| `MIKROBOT_MODEL` | Model to use (default: `gpt-4o`) |
| `MIKROBOT_WORKSPACE` | Workspace directory path |
| `MIKROBOT_WORK_REPO_PATH` | Work repo directory path |

---

## 9. FAQ / Troubleshooting

### "Config not found"

The configuration file does not exist yet. Run the onboard command to create it:

```bash
gomikrobot onboard
```

This creates `~/.gomikrobot/config.json` with default settings.

### "API Key not found"

The bot cannot find an LLM API key. Set it via environment variable or in the config file:

```bash
# Option 1: Environment variable
export OPENAI_API_KEY="sk-..."

# Option 2: Edit config.json
# Set the "apiKey" field under providers.openai in ~/.gomikrobot/config.json
```

The agent checks for keys in this order: `MIKROBOT_OPENAI_API_KEY`, `OPENROUTER_API_KEY`, then `config.json`.

### Port already in use

If ports 18790 or 18791 are occupied by a previous gateway process, use the `rerun` Makefile target which automatically kills existing processes:

```bash
cd gomikrobot
make rerun
```

This runs `lsof` to find and kill processes on ports 18790 and 18791, then rebuilds and starts the gateway.

### WhatsApp QR code not showing

The QR code is saved as an image file rather than displayed in the terminal. Check:

```
~/.gomikrobot/whatsapp-qr.png
```

Open this file with an image viewer and scan it with your phone.

If the file does not exist, ensure WhatsApp is enabled in your configuration:

```bash
gomikrobot whatsapp-setup
# Answer "y" to enable WhatsApp
```

### WhatsApp session already linked

If you see "Session found (no QR needed)" in `gomikrobot status`, your device is already linked. The session is stored in `~/.gomikrobot/whatsapp.db`. To re-link, delete this file and restart the gateway.

### Daily token quota exceeded

If you see the message "Daily token quota exceeded", it means the configured daily token limit has been reached. Options:

- **Wait until tomorrow** -- the quota resets daily
- **Increase the limit** -- set the `daily_token_limit` runtime setting via the web dashboard Settings panel or the `/api/v1/settings` API
- **Remove the limit** -- delete or clear the `daily_token_limit` setting to disable quota enforcement

### Messages not being delivered (silent mode)

If outbound messages are not arriving via WhatsApp, check if silent mode is enabled:

1. Open the web dashboard at `http://localhost:18791`
2. Check the Settings panel for `silent_mode`
3. Disable it, or enable `force_send` for specific web users who should receive messages regardless

### Agent returns "Max iterations reached"

The agent has a configurable limit on tool call iterations per request (default: 20). If the agent hits this limit, simplify your request or increase `maxToolIterations` in `config.json`.

### Docker Deployment

GoMikroBot can also be run via Docker:

```bash
cd gomikrobot

# Build and start
make docker-up

# View logs
make docker-logs

# Stop
make docker-down
```

The Docker setup expects `SYSTEM_REPO_PATH` and `WORK_REPO_PATH` environment variables (defaults to the current directory and `~/GoMikroBot-Workspace` respectively).

---

*GoMikroBot v1.1.0 -- A lightweight, ultra-fast AI assistant framework written in Go.*
