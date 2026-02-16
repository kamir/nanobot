#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# step1-SPEC-DOC-TASKS.sh
# 1. Removes the private submodule from KafClaw (no more public link to
#    the scalytics repo).
# 2. Adds private/ to .gitignore so nothing in it gets committed publicly.
# 3. Copies all specs, tasks, research, docs, and governance from nanobot/
#    into KafClaw/private/.
# Idempotent — safe to re-run.
# =============================================================================

SRC="/Users/kamir/GITHUB.kamir/nanobot"
DST="/Users/kamir/GITHUB.kamir/KafClaw"
PRIVATE="$DST/private"

# ---------------------------------------------------------------------------
# Pre-flight checks
# ---------------------------------------------------------------------------
if [[ ! -d "$SRC" ]]; then
  echo "ERROR: Source repo not found at $SRC" >&2; exit 1
fi
if [[ ! -d "$DST/.git" ]]; then
  echo "ERROR: KafClaw repo not found at $DST" >&2; exit 1
fi

echo "=== Step 1: Migrating specs, docs, tasks, research, governance ==="
echo "  Source: $SRC"
echo "  Target: $PRIVATE"
echo ""

# ---------------------------------------------------------------------------
# Remove the private submodule (if it still exists)
# ---------------------------------------------------------------------------
echo "--- Removing private submodule (if present) ---"

cd "$DST"

if [[ -f "$DST/.gitmodules" ]] && grep -q '\[submodule "private"\]' "$DST/.gitmodules" 2>/dev/null; then
  echo "  Deiniting submodule 'private'..."

  # Unregister the submodule from git config
  git submodule deinit -f private 2>/dev/null || true

  # Remove from the index
  git rm -f private 2>/dev/null || true

  # Remove cached module data
  rm -rf "$DST/.git/modules/private" 2>/dev/null || true

  # Remove .gitmodules if it's now empty (only had the one submodule)
  if [[ -f "$DST/.gitmodules" ]]; then
    if [[ ! -s "$DST/.gitmodules" ]] || ! grep -q '\[submodule' "$DST/.gitmodules" 2>/dev/null; then
      git rm -f .gitmodules 2>/dev/null || rm -f .gitmodules
      echo "  Removed empty .gitmodules"
    else
      echo "  .gitmodules still has other submodules — kept"
    fi
  fi

  echo "  Submodule removed."
else
  echo "  No submodule found — skipping removal."
fi

# Clean up any leftover private/ directory from the submodule
if [[ -d "$PRIVATE" ]] && [[ -f "$PRIVATE/.git" ]]; then
  echo "  Cleaning up leftover submodule worktree..."
  rm -rf "$PRIVATE"
fi

echo ""

# ---------------------------------------------------------------------------
# Add private/ to .gitignore
# ---------------------------------------------------------------------------
echo "--- Ensuring private/ is in .gitignore ---"
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

echo ""

# ---------------------------------------------------------------------------
# Create target directory tree
# ---------------------------------------------------------------------------
mkdir -p \
  "$PRIVATE/specs/architecture" \
  "$PRIVATE/specs/requirements" \
  "$PRIVATE/specs/design" \
  "$PRIVATE/tasks/devtasks" \
  "$PRIVATE/tasks/migration" \
  "$PRIVATE/tasks/inspiration" \
  "$PRIVATE/tasks/bug-tracker" \
  "$PRIVATE/research/hardening" \
  "$PRIVATE/research/drafts" \
  "$PRIVATE/docs/guides" \
  "$PRIVATE/docs/rebranding" \
  "$PRIVATE/governance/workspace"

echo "--- Target directories created ---"
echo ""

# ---------------------------------------------------------------------------
# specs/architecture/ ← gomikrobot/SPEC/ (9 files) + morphing/arch/ (5 files)
# ---------------------------------------------------------------------------
echo "--- specs/architecture/ ---"
cp -v "$SRC"/gomikrobot/SPEC/*.md "$PRIVATE/specs/architecture/" 2>/dev/null || echo "  (no files in gomikrobot/SPEC/)"
cp -v "$SRC"/morphing/arch/*.md   "$PRIVATE/specs/architecture/" 2>/dev/null || echo "  (no files in morphing/arch/)"
echo ""

# ---------------------------------------------------------------------------
# specs/requirements/ ← morphing/requirements/ (4 files)
# ---------------------------------------------------------------------------
echo "--- specs/requirements/ ---"
cp -v "$SRC"/morphing/requirements/*.md "$PRIVATE/specs/requirements/" 2>/dev/null || echo "  (no files in morphing/requirements/)"
echo ""

# ---------------------------------------------------------------------------
# specs/design/ ← morphing/docs/ (3 files)
# ---------------------------------------------------------------------------
echo "--- specs/design/ ---"
cp -v "$SRC"/morphing/docs/*.md "$PRIVATE/specs/design/" 2>/dev/null || echo "  (no files in morphing/docs/)"
echo ""

# ---------------------------------------------------------------------------
# tasks/devtasks/ ← DEVTASKS/*.md (3) + gomikrobot/DEVTASKS/*.md (2)
# ---------------------------------------------------------------------------
echo "--- tasks/devtasks/ ---"
cp -v "$SRC"/DEVTASKS/*.md           "$PRIVATE/tasks/devtasks/" 2>/dev/null || echo "  (no .md files in DEVTASKS/)"
cp -v "$SRC"/gomikrobot/DEVTASKS/*.md "$PRIVATE/tasks/devtasks/" 2>/dev/null || echo "  (no .md files in gomikrobot/DEVTASKS/)"
echo ""

# ---------------------------------------------------------------------------
# tasks/inspiration/ ← DEVTASKS/INSPIRATION/ (2 files)
# ---------------------------------------------------------------------------
echo "--- tasks/inspiration/ ---"
cp -v "$SRC"/DEVTASKS/INSPIRATION/* "$PRIVATE/tasks/inspiration/" 2>/dev/null || echo "  (no files in DEVTASKS/INSPIRATION/)"
echo ""

# ---------------------------------------------------------------------------
# tasks/migration/ ← morphing/tasks/ (3 files)
# ---------------------------------------------------------------------------
echo "--- tasks/migration/ ---"
cp -v "$SRC"/morphing/tasks/*.md "$PRIVATE/tasks/migration/" 2>/dev/null || echo "  (no files in morphing/tasks/)"
echo ""

# ---------------------------------------------------------------------------
# tasks/bug-tracker/ ← morphing/bug-tracker/ (may be empty)
# ---------------------------------------------------------------------------
echo "--- tasks/bug-tracker/ ---"
if ls "$SRC"/morphing/bug-tracker/* &>/dev/null; then
  cp -v "$SRC"/morphing/bug-tracker/* "$PRIVATE/tasks/bug-tracker/"
else
  echo "  (bug-tracker is empty — directory created)"
fi
echo ""

# ---------------------------------------------------------------------------
# research/hardening/ ← hardening/ (entire tree)
# ---------------------------------------------------------------------------
echo "--- research/hardening/ ---"
rsync -av --exclude='.DS_Store' "$SRC/hardening/" "$PRIVATE/research/hardening/"
echo ""

# ---------------------------------------------------------------------------
# research/drafts/ ← research-drafts/ (1 PDF)
# ---------------------------------------------------------------------------
echo "--- research/drafts/ ---"
cp -v "$SRC"/research-drafts/* "$PRIVATE/research/drafts/" 2>/dev/null || echo "  (no files in research-drafts/)"
echo ""

# ---------------------------------------------------------------------------
# docs/guides/ ← docs/*.md (15 files) + ARCHITECTURE-GO.md + MEMORY-GO.md
# ---------------------------------------------------------------------------
echo "--- docs/guides/ ---"
cp -v "$SRC"/docs/*.md "$PRIVATE/docs/guides/" 2>/dev/null || echo "  (no .md files in docs/)"
cp -v "$SRC/gomikrobot/ARCHITECTURE.md" "$PRIVATE/docs/guides/ARCHITECTURE-GO.md"
cp -v "$SRC/gomikrobot/MEMORY.md"       "$PRIVATE/docs/guides/MEMORY-GO.md"
echo ""

# ---------------------------------------------------------------------------
# docs/rebranding/ ← rebranding/ (all files)
# ---------------------------------------------------------------------------
echo "--- docs/rebranding/ ---"
rsync -av --exclude='.DS_Store' "$SRC/rebranding/" "$PRIVATE/docs/rebranding/"
echo ""

# ---------------------------------------------------------------------------
# governance/ ← AGENTS.md, CLAUDE.md (archived), workspace/
# ---------------------------------------------------------------------------
echo "--- governance/ ---"
cp -v "$SRC/AGENTS.md" "$PRIVATE/governance/AGENTS.md"
cp -v "$SRC/CLAUDE.md" "$PRIVATE/governance/CLAUDE-NANOBOT.md"
echo ""

echo "--- governance/workspace/ ---"
rsync -av --exclude='.DS_Store' "$SRC/workspace/" "$PRIVATE/governance/workspace/"
echo ""

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo "============================================="
echo " Step 1 complete!"
echo "============================================="
echo ""
echo "What happened:"
echo "  - Submodule 'private' removed (no more public link to scalytics repo)"
echo "  - private/ added to .gitignore (content stays local, never pushed publicly)"
echo "  - All specs/docs/tasks/research/governance copied into $PRIVATE/"
echo ""
echo "Next steps — commit submodule removal + .gitignore update to KafClaw:"
echo "  cd $DST"
echo "  git add .gitignore"
echo "  git status   # verify .gitmodules removal is staged, private/ is ignored"
echo "  git commit -m 'Remove private submodule, gitignore private/'"
echo "  git push origin main"
echo ""
echo "Then sync private/ content to the standalone private repo:"
echo "  ./migration/sync-private.sh"
echo ""
