#!/bin/sh
# One-click dev startup: Go backend (:3000) + Vite frontend (:8080).
set -e
cd "$(dirname "$0")"

command -v go >/dev/null 2>&1 || { echo "error: go not found in PATH"; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "error: npm not found in PATH"; exit 1; }

if [ ! -d web/node_modules ]; then
  echo "==> installing frontend dependencies"
  (cd web && npm install --no-fund --no-audit)
fi

BACKEND=""
FRONTEND=""
cleanup() {
  [ -n "$BACKEND" ] && kill "$BACKEND" 2>/dev/null
  [ -n "$FRONTEND" ] && kill "$FRONTEND" 2>/dev/null
}
trap cleanup EXIT INT TERM

echo "==> starting backend on :3000"
# Fixed dev secret so backend restarts don't invalidate login tokens.
if [ -z "$MC_JWT_SECRET" ]; then export MC_JWT_SECRET=mcdev-insecure-jwt-secret; fi
go run ./cmd/server &
BACKEND=$!

echo "==> starting frontend on :8080"
(cd web && npm run dev) &
FRONTEND=$!

echo "==> ready: http://localhost:8080"
wait
