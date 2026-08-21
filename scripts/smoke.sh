#!/usr/bin/env bash
set -euo pipefail

BASE="${BASE_URL:-http://localhost:8080}"

echo "healthz:"
curl -fsS "$BASE/healthz"
echo
echo "readyz:"
curl -fsS "$BASE/readyz"
echo
echo "metrics sample:"
curl -fsS "$BASE/metrics" | head -n 12

echo "create rule:"
RULE_JSON='{"name":"cpu-high","data_source":"demo-api","event_type":"metric","type":"threshold","condition":{"field":"avg","operator":"gt","value":0.7},"severity":"critical","labels":{"team":"platform"},"group_by":["region"],"channels":[]}'
curl -fsS -X POST "$BASE/api/v1/rules" -H 'Content-Type: application/json' -d "$RULE_JSON"
echo

echo "ingest batch:"
INGEST_JSON='{"source":"demo-api","events":[{"type":"metric","labels":{"region":"cn-north"},"numeric_value":0.92,"occurred_at":"2026-08-19T12:00:00Z"}]}'
curl -fsS -X POST "$BASE/api/v1/ingest/batch" -H 'Content-Type: application/json' -d "$INGEST_JSON"
echo

echo "evaluate:"
curl -fsS -X POST "$BASE/api/v1/evaluate"
echo

echo "alerts:"
curl -fsS "$BASE/api/v1/alerts"
echo
