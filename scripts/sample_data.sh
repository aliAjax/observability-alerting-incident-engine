#!/usr/bin/env bash
set -euo pipefail

BASE="${BASE_URL:-http://localhost:8080}"

curl -fsS -X POST "$BASE/api/v1/rules" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "api-latency-spike",
    "data_source": "demo-api",
    "event_type": "metric",
    "type": "threshold",
    "condition": {"field": "avg", "operator": "gt", "value": 0.85},
    "severity": "critical",
    "labels": {"team": "platform"},
    "group_by": ["region"],
    "channels": []
  }'

for value in 0.62 0.71 0.88 0.94 0.81 0.97; do
  curl -fsS -X POST "$BASE/api/v1/ingest/batch" \
    -H 'Content-Type: application/json' \
    -d "{\"source\":\"demo-api\",\"events\":[{\"type\":\"metric\",\"labels\":{\"region\":\"cn-north\"},\"numeric_value\":$value,\"occurred_at\":\"2026-08-19T12:00:00Z\"}]}" >/dev/null
done

curl -fsS -X POST "$BASE/api/v1/evaluate"
curl -fsS "$BASE/api/v1/alerts"
