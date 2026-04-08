#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_FILE="${1:-$ROOT_DIR/test/test.srt}"

cd "$ROOT_DIR"

go run . transcript \
  --file test/test.mp3 \
  --output "$OUT_FILE" \
  --source-lang zh \
  --target-lang en

echo
echo "Preview:"
sed -n '1,40p' "$OUT_FILE"
