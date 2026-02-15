# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GoMikroBot is a personal AI assistant being rewritten from Python to Go. The Go code lives in `gomikrobot/`. The Python code in `nanobot/` is the legacy implementation being replaced. The migration follows a behavior-first approach: parity before optimization.

## Build & Run

All Go commands run from the `gomikrobot/` directory:

```bash
cd gomikrobot

# Build
make build                    # or: go build ./cmd/gomikrobot

# Run gateway (multi-channel daemon)
make run                      # build + run
make rerun                    # kill existing ports 18790/18791, rebuild, run

# Run single message
./gomikrobot agent -m "hello"

# Run tests
go test ./...                 # all tests
go test ./internal/tools/     # single package

# Install to /usr/local/bin
make install

# Release (bump version + build)
make release-patch            # or release-minor, release-major
```

**Go version:** 1.24.0+ (toolchain 1.24.13)

## Architecture

```
CLI/WhatsApp → Message Bus → Agent Loop → LLM Provider (OpenAI/OpenRouter)
                                ↓
                           Tool Registry → Filesystem / Shell / Web
                                ↑
                           Context Builder (loads soul files from workspace/)
```

### Key packages (`gomikrobot/internal/`)

- **agent/** — Core agent loop (`loop.go`) and context/soul-file loader (`context.go`). The context builder assembles system prompts from `AGENTS.md`, `SOUL.md`, `USER.md`, `TOOLS.md`, `IDENTITY.md` in the workspace directory.
- **bus/** — Async message bus decoupling channels from the agent loop (pub-sub).
- **channels/** — External integrations. WhatsApp uses `whatsmeow` (native, no Node bridge). Implements `Channel` interface.
- **config/** — Config struct with env/file/default loading. Config file: `~/.gomikrobot/config.json`. Env prefix: `MIKROBOT_`.
- **provider/** — LLM provider abstraction. OpenAI/OpenRouter implementations, Whisper transcription, TTS.
- **session/** — Per-session conversation history, JSONL persistence, thread-safe.
- **timeline/** — SQLite event log at `~/.gomikrobot/timeline.db`. Stores messages, settings, web user mappings. Has trace/span IDs.
- **tools/** — Registry-based tool system. Filesystem ops have path safety; shell exec has deny-pattern filtering and timeout (default 60s).

### CLI commands (`gomikrobot/cmd/gomikrobot/cmd/`)

- `gateway` — Main daemon: WhatsApp + CLI + Web UI on ports 18790/18791
- `agent -m "msg"` — Single-message mode
- `onboard` — First-time setup
- `status` — Version and config check
- `whatsapp-setup` / `whatsapp-auth` — WhatsApp configuration
- `install` — System install

## Configuration

Loaded in order: env vars > `~/.gomikrobot/config.json` > defaults.

Default model: `gpt-4o`. Default workspace: `~/.gomikrobot/workspace`. Default work repo: `~/.gomikrobot/work-repo`. Gateway ports: 18790 (API), 18791 (dashboard).

Runtime settings (bot_repo_path, whatsapp_allowlist, etc.) are stored in the timeline SQLite database.

## Migration Governance (from AGENTS.md)

- **Source of truth flows one way:** `PH/ → requirements/ → arch/ → DEVTASKS/`
- PH/ (Prompt History) is immutable — never modify it
- Any CLI command/flag change must be documented in `/docs` immediately
- Behavior parity first, improvements second
- No task should exceed 2 days; each needs acceptance criteria and a rollback path

## Tool Security Model

Shell execution (`internal/tools/shell.go`) uses deny-pattern filtering (blocks `rm`, `chmod`, `mkfs`, `shutdown`, fork bombs, etc.) and allow-pattern lists in strict mode. Filesystem writes are restricted to the work repo by default. Path traversal (`../`) is blocked.

## Extending the System

**New tool:** Implement the `Tool` interface in `internal/tools/` (Name, Description, Parameters, Execute methods), then register in the agent loop's `registerDefaultTools()`.

**New channel:** Implement `Channel` interface in `internal/channels/`, subscribe to the message bus, add config fields to `internal/config/config.go`.

**New CLI command:** Create file in `cmd/gomikrobot/cmd/`, define cobra command, register in `root.go` init().
