#!/usr/bin/env bash
# Smoke checks against a running API (see docs/STAGING_SMOKE_TEST.md).
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8080}"
ADMIN_PRINCIPAL="${ADMIN_PRINCIPAL:-30000000-0000-0000-0000-000000000001}"

echo "==> GET $API_BASE/health"
curl -sfS "$API_BASE/health" | head -c 200
echo
echo "==> GET $API_BASE/ops/health"
curl -sfS "$API_BASE/ops/health" | head -c 400
echo
echo "==> GET $API_BASE/domains (X-Principal-User-ID)"
curl -sfS -H "X-Principal-User-ID: $ADMIN_PRINCIPAL" "$API_BASE/domains" | head -c 400
echo
echo "==> GET $API_BASE/search?q=test (X-Principal-User-ID)"
curl -sfS -H "X-Principal-User-ID: $ADMIN_PRINCIPAL" "$API_BASE/search?q=test" | head -c 400
echo
echo "==> GET $API_BASE/knowledge-jobs/engine-metadata (X-Principal-User-ID)"
curl -sfS -H "X-Principal-User-ID: $ADMIN_PRINCIPAL" "$API_BASE/knowledge-jobs/engine-metadata" | head -c 400
echo
# Warm a labeled series for the custom Prometheus counter (it won't appear in /metrics until observed).
curl -sfS "$API_BASE/health" >/dev/null

echo "==> GET $API_BASE/metrics (Prometheus; bearer if OPS_AUTH_TOKEN set)"
if [ -n "${OPS_AUTH_TOKEN:-}" ]; then
  METRICS_FILTERED="$(
    curl -sfS -H "Authorization: Bearer $OPS_AUTH_TOKEN" "$API_BASE/metrics" \
      | grep -E '^# HELP http_requests_total|^# TYPE http_requests_total|http_requests_total\{' \
      | head -c 1200
  )"
else
  METRICS_FILTERED="$(
    curl -sfS "$API_BASE/metrics" \
      | grep -E '^# HELP http_requests_total|^# TYPE http_requests_total|http_requests_total\{' \
      | head -c 1200
  )"
fi
echo "$METRICS_FILTERED"
echo
if ! echo "$METRICS_FILTERED" | grep -q 'http_requests_total{'; then
  echo "smoke-local: expected http_requests_total series in /metrics output" >&2
  exit 1
fi
echo "smoke-local: ok"
