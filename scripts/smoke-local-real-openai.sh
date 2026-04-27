#!/usr/bin/env bash
# Smoke checks for local stack + optional real OpenAI Ask (see docs/GETTING_STARTED.md).
# - Always: /health, /domains, /search (needs OpenSearch up).
# - POST /ask: runs when OPENAI_API_KEY or OPENROUTER_API_KEY is set and OPENAI_MOCK is not 1 (may incur API usage).
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8080}"
ADMIN_PRINCIPAL="${ADMIN_PRINCIPAL:-30000000-0000-0000-0000-000000000001}"

echo "==> GET $API_BASE/health"
curl -sfS "$API_BASE/health" | head -c 200
echo

echo "==> GET $API_BASE/domains (X-Principal-User-ID)"
DOMAINS_JSON="$(curl -sfS -H "X-Principal-User-ID: $ADMIN_PRINCIPAL" "$API_BASE/domains")"
echo "$DOMAINS_JSON" | head -c 800
echo

DOMAIN_ID="$(echo "$DOMAINS_JSON" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    rows = data if isinstance(data, list) else data.get('domains') or data.get('items') or []
    if rows and isinstance(rows[0], dict):
        rid = rows[0].get('id') or rows[0].get('domain_id')
        if rid:
            print(rid)
except Exception:
    pass
" 2>/dev/null || true)"

echo "==> GET $API_BASE/search?q=test (X-Principal-User-ID)"
curl -sfS -H "X-Principal-User-ID: $ADMIN_PRINCIPAL" "$API_BASE/search?q=test" | head -c 400
echo

if { [ -z "${OPENAI_API_KEY:-}" ] && [ -z "${OPENROUTER_API_KEY:-}" ]; } || [ "${OPENAI_MOCK:-}" = "1" ]; then
  echo "smoke-local-real-openai: skipping POST /ask (set OPENAI_API_KEY or OPENROUTER_API_KEY; unset OPENAI_MOCK for real LLM smoke)"
  echo "smoke-local-real-openai: ok (infra + search)"
  exit 0
fi

if [ -z "${DOMAIN_ID:-}" ]; then
  echo "smoke-local-real-openai: no domain id parsed; open /bootstrap or ensure AUTO_BOOTSTRAP_INSTANCE" >&2
  exit 1
fi

echo "==> POST $API_BASE/ask (real OpenAI when key set; domain_id=$DOMAIN_ID)"
ASK_BODY="$(curl -sS -X POST "$API_BASE/ask" \
  -H "Content-Type: application/json" \
  -H "X-Principal-User-ID: $ADMIN_PRINCIPAL" \
  -d "{\"question\":\"What is in the knowledge base?\",\"domain_id\":\"$DOMAIN_ID\"}" \
  -w '\n%{http_code}' )"
ASK_HTTP="$(echo "$ASK_BODY" | tail -n1)"
ASK_TEXT="$(echo "$ASK_BODY" | sed '$d')"
echo "$ASK_TEXT" | head -c 1200
echo
if [ "$ASK_HTTP" = "200" ]; then
  echo "smoke-local-real-openai: ok (Ask returned 200)"
  exit 0
fi
if [ "$ASK_HTTP" = "400" ] && echo "$ASK_TEXT" | grep -q 'no search evidence'; then
  echo "smoke-local-real-openai: ok (Ask skipped: empty KB / no retrieval hits — add entities or broaden query for full Ask smoke)"
  exit 0
fi
echo "smoke-local-real-openai: unexpected Ask HTTP $ASK_HTTP" >&2
exit 1
