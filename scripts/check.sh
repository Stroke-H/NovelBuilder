#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

echo "[check] apps/api -> go test ./..."
(
  cd "$ROOT_DIR/apps/api"
  go test ./...
)

echo "[check] apps/web -> npm run build"
(
  cd "$ROOT_DIR/apps/web"
  npm run build
)
