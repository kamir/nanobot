#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROOT_GO="${ROOT_DIR}/cmd/gomikrobot/cmd/root.go"

if [[ ! -f "$ROOT_GO" ]]; then
  echo "root.go not found: $ROOT_GO" >&2
  exit 1
fi

PART="${1:-}"
if [[ "$PART" != "major" && "$PART" != "minor" && "$PART" != "patch" ]]; then
  echo "Usage: $0 {major|minor|patch}" >&2
  exit 1
fi

CURRENT="$(grep -n 'version = "' "$ROOT_GO" | head -n1 | sed -E 's/.*version = \"([0-9]+\.[0-9]+\.[0-9]+)\".*/\1/')"
if [[ ! "$CURRENT" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Failed to parse version from $ROOT_GO" >&2
  exit 1
fi

IFS='.' read -r MAJOR MINOR PATCH <<<"$CURRENT"

case "$PART" in
  major)
    MAJOR=$((MAJOR + 1))
    MINOR=0
    PATCH=0
    ;;
  minor)
    MINOR=$((MINOR + 1))
    PATCH=0
    ;;
  patch)
    PATCH=$((PATCH + 1))
    ;;
esac

NEXT="${MAJOR}.${MINOR}.${PATCH}"

perl -0777 -i -pe "s/version = \\\"$CURRENT\\\"/version = \\\"$NEXT\\\"/g" "$ROOT_GO"

echo "Version bumped: $CURRENT -> $NEXT"

git add "$ROOT_GO"
git commit -m "Release v$NEXT"
git tag "v$NEXT"
git push
git push --tags
