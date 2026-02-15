#!/usr/bin/env bash
set -euo pipefail

BIN_SOURCE="${1:-}"
if [[ -z "$BIN_SOURCE" ]]; then
  BIN_SOURCE="$(pwd)/gomikrobot"
fi

TARGET_DIR="/usr/local/bin"
TARGET_PATH="${TARGET_DIR}/gomikrobot"

if [[ ! -f "$BIN_SOURCE" ]]; then
  echo "Binary not found: $BIN_SOURCE" >&2
  echo "Build it first: make build" >&2
  exit 1
fi

mkdir -p "$TARGET_DIR" 2>/dev/null || true

if cp "$BIN_SOURCE" "$TARGET_PATH" 2>/dev/null; then
  echo "Installed to $TARGET_PATH"
  exit 0
fi

echo "Install requires sudo:"
echo "sudo cp \"$BIN_SOURCE\" \"$TARGET_PATH\""
