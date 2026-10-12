#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
echo "==> building frontend"
(cd web && npm ci && npm run build)   # -> web/dist
echo "==> building linux binary"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o dist/pvdiary ./cmd/pvdiary
echo "==> done: dist/pvdiary"
