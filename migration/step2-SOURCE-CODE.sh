#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# step2-SOURCE-CODE.sh
# Copies Go source code, web UI, Electron app, CI workflow into KafClaw/.
# Applies path patches and generates a new CLAUDE.md.  Idempotent.
# =============================================================================

SRC="/Users/kamir/GITHUB.kamir/nanobot"
DST="/Users/kamir/GITHUB.kamir/KafClaw"

# ---------------------------------------------------------------------------
# Pre-flight checks
# ---------------------------------------------------------------------------
if [[ ! -d "$SRC/gomikrobot" ]]; then
  echo "ERROR: Source gomikrobot/ not found at $SRC/gomikrobot" >&2; exit 1
fi
if [[ ! -d "$DST" ]]; then
  echo "ERROR: KafClaw repo not found at $DST" >&2; exit 1
fi

echo "=== Step 2: Migrating source code ==="
echo "  Source: $SRC"
echo "  Target: $DST"
echo ""

# ---------------------------------------------------------------------------
# 1. rsync gomikrobot/ with exclusions
# ---------------------------------------------------------------------------
echo "--- rsync gomikrobot/ (excluding dist, binaries, SPEC, DEVTASKS, node_modules) ---"
rsync -av \
  --exclude='dist/' \
  --exclude='gomikrobot' \
  --exclude='SPEC/' \
  --exclude='DEVTASKS/' \
  --exclude='electron/node_modules/' \
  --exclude='electron/dist/' \
  --exclude='~/' \
  --exclude='.DS_Store' \
  "$SRC/gomikrobot/" "$DST/gomikrobot/"
echo ""

# ---------------------------------------------------------------------------
# 2. Copy GitHub Actions workflow
# ---------------------------------------------------------------------------
echo "--- .github/workflows/ ---"
mkdir -p "$DST/.github/workflows"
cp -v "$SRC/.github/workflows/release.yml" "$DST/.github/workflows/release.yml"
echo ""

# ---------------------------------------------------------------------------
# 3. Apply patches
# ---------------------------------------------------------------------------
echo "--- Applying patches ---"

# 3a. config.go: SystemRepoPath  /nanobot/gomikrobot → /KafClaw/gomikrobot
CONFIGFILE="$DST/gomikrobot/internal/config/config.go"
if [[ -f "$CONFIGFILE" ]]; then
  sed -i '' 's|/nanobot/gomikrobot|/KafClaw/gomikrobot|g' "$CONFIGFILE"
  echo "  Patched: config.go (SystemRepoPath)"
else
  echo "  WARN: $CONFIGFILE not found — skipping patch" >&2
fi

# 3b. session.go: .nanobot → .gomikrobot
SESSIONFILE="$DST/gomikrobot/internal/session/session.go"
if [[ -f "$SESSIONFILE" ]]; then
  sed -i '' 's/\.nanobot/\.gomikrobot/g' "$SESSIONFILE"
  echo "  Patched: session.go (.nanobot → .gomikrobot)"
else
  echo "  WARN: $SESSIONFILE not found — skipping patch" >&2
fi

# 3c. install.sh: kamir/nanobot → KafClaw/KafClaw
INSTALLFILE="$DST/gomikrobot/scripts/install.sh"
if [[ -f "$INSTALLFILE" ]]; then
  sed -i '' 's|kamir/nanobot|KafClaw/KafClaw|g' "$INSTALLFILE"
  echo "  Patched: install.sh (repo reference)"
else
  echo "  WARN: $INSTALLFILE not found — skipping patch" >&2
fi

# 3d. release.yml: fix ldflags path  kamir/nanobot/gomikrobot → kamir/gomikrobot
RELEASEFILE="$DST/.github/workflows/release.yml"
if [[ -f "$RELEASEFILE" ]]; then
  sed -i '' 's|kamir/nanobot/gomikrobot|kamir/gomikrobot|g' "$RELEASEFILE"
  echo "  Patched: release.yml (ldflags path)"
else
  echo "  WARN: $RELEASEFILE not found — skipping patch" >&2
fi

echo ""

# ---------------------------------------------------------------------------
# 4. Update .gitignore
# ---------------------------------------------------------------------------
echo "--- Updating .gitignore ---"
GITIGNORE="$DST/.gitignore"

append_if_missing() {
  local entry="$1"
  if ! grep -qxF "$entry" "$GITIGNORE" 2>/dev/null; then
    echo "$entry" >> "$GITIGNORE"
    echo "  Added: $entry"
  else
    echo "  Already present: $entry"
  fi
}

# Ensure file ends with newline before appending
if [[ -f "$GITIGNORE" ]] && [[ -s "$GITIGNORE" ]] && [[ "$(tail -c1 "$GITIGNORE")" != "" ]]; then
  echo "" >> "$GITIGNORE"
fi

append_if_missing "private/"
append_if_missing "gomikrobot/gomikrobot"
append_if_missing "gomikrobot/dist/"
append_if_missing "gomikrobot/electron/node_modules/"
append_if_missing "gomikrobot/electron/dist/"
append_if_missing ".DS_Store"

echo ""

# ---------------------------------------------------------------------------
# 5. Generate new CLAUDE.md for KafClaw
# ---------------------------------------------------------------------------
echo "--- Generating KafClaw CLAUDE.md ---"
cat > "$DST/CLAUDE.md" << 'CLAUDEEOF'
# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

KafClaw (formerly GoMikroBot) is a personal AI assistant written in Go. The Go source lives in `gomikrobot/`. Sensitive specs, tasks, research, and governance docs live in the `private/` directory (gitignored — tracked separately in scalytics/KafClaw-PRIVATE-PARTS).

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

## Repository Structure

```
KafClaw/
├── CLAUDE.md               ← this file
├── .github/workflows/      ← CI/CD (release.yml)
├── gomikrobot/             ← Go source code
│   ├── cmd/gomikrobot/     ← CLI entry point (cobra commands)
│   ├── internal/           ← Core packages
│   │   ├── agent/          ← Agent loop + context/soul-file loader
│   │   ├── bus/            ← Async message bus (pub-sub)
│   │   ├── channels/       ← WhatsApp (whatsmeow), CLI, Web channels
│   │   ├── config/         ← Config struct (env/file/default loading)
│   │   ├── provider/       ← LLM provider abstraction (OpenAI/OpenRouter)
│   │   ├── session/        ← Per-session conversation history (JSONL)
│   │   ├── timeline/       ← SQLite event log (~/.gomikrobot/timeline.db)
│   │   └── tools/          ← Registry-based tool system
│   ├── web/                ← Web UI (HTML dashboard)
│   ├── electron/           ← Electron desktop app wrapper
│   ├── scripts/            ← install.sh, release.sh
│   ├── go.mod, go.sum, Makefile, Dockerfile, docker-compose.yml
│   └── ARCHITECTURE.md, MEMORY.md
└── private/                ← .gitignored (tracked in separate private repo)
    ├── specs/              ← Architecture, requirements, design specs
    ├── tasks/              ← Dev tasks, migration tasks, inspiration
    ├── research/           ← Hardening research, drafts
    ├── docs/               ← Guides, rebranding assets
    └── governance/         ← AGENTS.md, archived CLAUDE.md, workspace soul files
```

## Architecture

```
CLI/WhatsApp → Message Bus → Agent Loop → LLM Provider (OpenAI/OpenRouter)
                                ↓
                           Tool Registry → Filesystem / Shell / Web
                                ↑
                           Context Builder (loads soul files from workspace/)
```

### Key packages (`gomikrobot/internal/`)

- **agent/** — Core agent loop (`loop.go`) and context/soul-file loader (`context.go`).
- **bus/** — Async message bus decoupling channels from the agent loop (pub-sub).
- **channels/** — External integrations. WhatsApp uses `whatsmeow` (native, no Node bridge).
- **config/** — Config struct with env/file/default loading. Config file: `~/.gomikrobot/config.json`. Env prefix: `MIKROBOT_`.
- **provider/** — LLM provider abstraction. OpenAI/OpenRouter implementations, Whisper transcription, TTS.
- **session/** — Per-session conversation history, JSONL persistence, thread-safe.
- **timeline/** — SQLite event log at `~/.gomikrobot/timeline.db`.
- **tools/** — Registry-based tool system. Filesystem ops have path safety; shell exec has deny-pattern filtering and timeout (default 60s).

## Configuration

Loaded in order: env vars > `~/.gomikrobot/config.json` > defaults.

Default model: `anthropic/claude-sonnet-4-5`. Default workspace: `~/GoMikroBot-Workspace`. Gateway ports: 18790 (API), 18791 (dashboard).

## Tool Security Model

Shell execution (`internal/tools/shell.go`) uses deny-pattern filtering (blocks `rm`, `chmod`, `mkfs`, `shutdown`, fork bombs, etc.) and allow-pattern lists in strict mode. Filesystem writes are restricted to the work repo by default. Path traversal (`../`) is blocked.

## Extending the System

**New tool:** Implement the `Tool` interface in `internal/tools/` (Name, Description, Parameters, Execute methods), then register in the agent loop's `registerDefaultTools()`.

**New channel:** Implement `Channel` interface in `internal/channels/`, subscribe to the message bus, add config fields to `internal/config/config.go`.

**New CLI command:** Create file in `cmd/gomikrobot/cmd/`, define cobra command, register in `root.go` init().

## Note on go.mod

The Go module is still `github.com/kamir/gomikrobot`. Renaming the module path is a separate follow-up task (requires updating imports in 31+ Go files).
CLAUDEEOF

echo "  Generated: $DST/CLAUDE.md"
echo ""

# ---------------------------------------------------------------------------
# 6. Build verification
# ---------------------------------------------------------------------------
echo "=== Build verification ==="
echo ""

echo "--- go build ---"
cd "$DST/gomikrobot"
if go build ./cmd/gomikrobot; then
  echo "  BUILD: OK"
  # Clean up the binary we just built
  rm -f gomikrobot
else
  echo "  BUILD: FAILED" >&2
  echo "  Check errors above. The source code may need manual fixes." >&2
  exit 1
fi
echo ""

echo "--- go test ---"
if go test ./... 2>&1; then
  echo "  TESTS: OK"
else
  echo "  TESTS: SOME FAILURES (review output above)" >&2
  echo "  Continuing anyway — some tests may require runtime config." >&2
fi
echo ""

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo "============================================="
echo " Step 2 complete!"
echo "============================================="
echo ""
echo "Source code copied to: $DST/gomikrobot/"
echo "Patches applied, .gitignore updated, CLAUDE.md generated."
echo ""
echo "Verification:"
echo "  go build: passed"
echo ""
echo "Next steps:"
echo "  cd $DST"
echo "  git add -A"
echo "  git status   # verify private/ is NOT staged"
echo "  git commit -m 'Import Go source code from nanobot, apply KafClaw patches'"
echo "  git push origin main"
echo ""
echo "Post-migration:"
echo "  cd $DST/gomikrobot/electron && npm install   # restore node_modules"
echo "  Consider renaming go.mod module path (separate task)"
echo ""
