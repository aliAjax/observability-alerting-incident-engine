#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

docker compose up -d postgres
echo "waiting for postgres..."
for _ in $(seq 1 30); do
  if docker compose exec -T postgres pg_isready -U observability -d observability >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

mkdir -p bin
go build -o bin/server ./cmd/server
bin/server configs/config.yaml
