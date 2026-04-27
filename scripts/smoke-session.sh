#!/usr/bin/env bash
# Session-based smoke checks (AUTH_MODE=session). See docs/STAGING_SMOKE_TEST.md § Session smoke.
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8080}"
SMOKE_EMAIL="${SMOKE_EMAIL:?set SMOKE_EMAIL (user with password_hash set)}"
SMOKE_PASSWORD="${SMOKE_PASSWORD:?set SMOKE_PASSWORD}"

COOKIE_JAR="$(mktemp)"
trap 'rm -f "$COOKIE_JAR"' EXIT

curl_opts=(-sfS)
if [[ "${CURL_INSECURE:-0}" == "1" ]]; then
  curl_opts+=(-k)
fi

echo "==> GET $API_BASE/health"
curl "${curl_opts[@]}" "$API_BASE/health" | head -c 200
echo

echo "==> GET $API_BASE/ops/health"
curl "${curl_opts[@]}" "$API_BASE/ops/health" | head -c 500
echo

echo "==> GET $API_BASE/metrics"
if [[ -n "${OPS_AUTH_TOKEN:-}" ]]; then
  METRICS_OUT="$(curl "${curl_opts[@]}" -H "Authorization: Bearer $OPS_AUTH_TOKEN" "$API_BASE/metrics")"
else
  METRICS_OUT="$(curl "${curl_opts[@]}" "$API_BASE/metrics" || true)"
fi
echo "$METRICS_OUT" | grep -E '^# HELP http_requests_total|^# TYPE http_requests_total|http_requests_total\{' | head -c 1200 || true
echo
if echo "$METRICS_OUT" | grep -q 'http_requests_total{'; then
  echo "smoke-session: http_requests_total present in /metrics"
else
  echo "smoke-session: warning — http_requests_total not found (non-local often needs OPS_AUTH_TOKEN + bearer)" >&2
fi

echo "==> POST $API_BASE/auth/login (cookie jar)"
login_json="$(python3 -c 'import json,sys; print(json.dumps({"email":sys.argv[1],"password":sys.argv[2]}))' "$SMOKE_EMAIL" "$SMOKE_PASSWORD")"
curl "${curl_opts[@]}" -c "$COOKIE_JAR" -H "Content-Type: application/json" \
  -d "$login_json" \
  "$API_BASE/auth/login" | head -c 400
echo

if ! grep -q 'kl_session' "$COOKIE_JAR" 2>/dev/null; then
  echo "smoke-session: expected kl_session cookie after login (wrong credentials or AUTH_MODE!=session?)" >&2
  exit 1
fi

echo "==> GET $API_BASE/domains (session cookie)"
curl "${curl_opts[@]}" -b "$COOKIE_JAR" "$API_BASE/domains" | head -c 600
echo

echo "==> GET $API_BASE/search?q=test (session cookie)"
curl "${curl_opts[@]}" -b "$COOKIE_JAR" "$API_BASE/search?q=test" | head -c 600
echo

echo "==> GET $API_BASE/knowledge-jobs/engine-metadata (session cookie)"
curl "${curl_opts[@]}" -b "$COOKIE_JAR" "$API_BASE/knowledge-jobs/engine-metadata" | head -c 600
echo

echo "==> GET $API_BASE/auth/me (session cookie)"
curl "${curl_opts[@]}" -b "$COOKIE_JAR" "$API_BASE/auth/me" | head -c 400
echo

echo "smoke-session: ok"
