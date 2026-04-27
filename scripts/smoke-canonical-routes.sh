#!/usr/bin/env bash
# Optional: verify Next.js redirects for canonical admin URLs (requires web running).
# Usage: WEB_BASE=http://localhost:13000 ./scripts/smoke-canonical-routes.sh
set -euo pipefail
WEB_BASE="${WEB_BASE:-http://localhost:13000}"

check_redirect() {
  local path="$1"
  local want="$2"
  local code
  code=$(curl -s -o /dev/null -w "%{http_code}" -L "$WEB_BASE$path" || true)
  if [[ "$code" != "$want" ]]; then
    echo "FAIL $path expected HTTP $want got $code" >&2
    return 1
  fi
  echo "OK $path -> $code"
}

# Permanent redirects from next.config (308 -> browser may show final 200 after follow -L)
# We follow redirects; final URL should land on CP or dash home.
check_redirect "/admin/roles" "200"
check_redirect "/access" "200"
check_redirect "/app/search" "200"

echo "smoke-canonical-routes: done (WEB_BASE=$WEB_BASE)"
