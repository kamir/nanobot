#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# sync-private.sh
# Syncs the contents of KafClaw/private/ to the standalone private repo
# (scalytics/KafClaw-PRIVATE-PARTS).  Idempotent — safe to re-run.
#
# The private repo is cloned alongside KafClaw if not already present.
# Content is rsynced from KafClaw/private/ → the private repo worktree,
# then staged.  You commit and push manually.
# =============================================================================

KAFCLAW="/Users/kamir/GITHUB.kamir/KafClaw"
PRIVATE_SRC="$KAFCLAW/private"
PRIVATE_REPO="/Users/kamir/GITHUB.kamir/KafClaw-PRIVATE-PARTS"
PRIVATE_REMOTE="https://github.com/scalytics/KafClaw-PRIVATE-PARTS"

# ---------------------------------------------------------------------------
# Pre-flight
# ---------------------------------------------------------------------------
if [[ ! -d "$PRIVATE_SRC" ]]; then
  echo "ERROR: $PRIVATE_SRC does not exist." >&2
  echo "  Run step1-SPEC-DOC-TASKS.sh first." >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# Clone the private repo if not already present
# ---------------------------------------------------------------------------
if [[ ! -d "$PRIVATE_REPO/.git" ]]; then
  echo "--- Cloning private repo ---"
  git clone "$PRIVATE_REMOTE" "$PRIVATE_REPO"
  echo ""
else
  echo "--- Private repo already present at $PRIVATE_REPO ---"
  echo "  Pulling latest..."
  cd "$PRIVATE_REPO"
  git pull --ff-only || echo "  (pull skipped — may have local changes)"
  echo ""
fi

# ---------------------------------------------------------------------------
# Sync content:  KafClaw/private/ → private repo worktree
# ---------------------------------------------------------------------------
echo "--- Syncing KafClaw/private/ → $PRIVATE_REPO/ ---"

# rsync with --delete so removed files are also removed in the private repo.
# Exclude .git so we don't clobber the private repo's own git data.
rsync -av --delete \
  --exclude='.git' \
  --exclude='.DS_Store' \
  "$PRIVATE_SRC/" "$PRIVATE_REPO/"

echo ""

# ---------------------------------------------------------------------------
# Stage changes
# ---------------------------------------------------------------------------
echo "--- Staging changes in private repo ---"
cd "$PRIVATE_REPO"
git add -A
echo ""

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo "============================================="
echo " Sync complete!"
echo "============================================="
echo ""
echo "Changes staged in: $PRIVATE_REPO"
echo ""
echo "Next steps:"
echo "  cd $PRIVATE_REPO"
echo "  git status"
echo "  git diff --cached --stat"
echo "  git commit -m 'Sync specs, docs, tasks, research, governance from KafClaw/private'"
echo "  git push origin main"
echo ""
