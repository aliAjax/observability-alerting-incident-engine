#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

pkill -f 'bin/server configs/config.yaml' 2>/dev/null || true
docker compose down
