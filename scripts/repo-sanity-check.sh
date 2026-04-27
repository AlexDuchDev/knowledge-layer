#!/usr/bin/env bash
# Heuristic checks before open-sourcing or pushing (secrets, tracked env files).
# Does not replace a full secret scanner; run from monorepo root: Knowledge Layer Local/
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
echo "repo-sanity-check: scanning from $ROOT"

if command -v git >/dev/null 2>&1 && git rev-parse --show-toplevel >/dev/null 2>&1; then
  if git ls-files -z -- '*.env' '*.pem' 2>/dev/null | grep -qz .; then
    echo "repo-sanity-check: FAIL — tracked .env or .pem files found in git index" >&2
    git ls-files '*.env' '*.pem' 2>/dev/null || true
    exit 1
  fi
  # Common high-entropy secret filenames (heuristic)
  if git ls-files | grep -E '(^|/)id_rsa$|\.key$' | grep -v node_modules | head -5 | grep -q .; then
    echo "repo-sanity-check: WARNING — private key-like paths in git; verify intent" >&2
  fi
else
  echo "repo-sanity-check: not a git checkout — skipping tracked-file checks"
fi

if [[ -f .env ]]; then
  echo "repo-sanity-check: WARNING — .env exists in working tree (must not be committed)" >&2
fi

echo "repo-sanity-check: ok"
