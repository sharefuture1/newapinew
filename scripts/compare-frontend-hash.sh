#!/usr/bin/env bash
# Compare the hashed index.*.js filename served by go.wending.ai and
# api.wending.ai. A mismatch means the Vercel frontend and the vultr-sv
# binary (//go:embed web/dist) are out of date with each other.
#
# Usage: scripts/compare-frontend-hash.sh
# Exit 0 when both sites serve the same hash, 1 on drift or a fetch failure.
set -euo pipefail

extract_index() {
  local url="$1"
  curl -fsSL --max-time 25 -H 'Accept: text/html' "$url" \
    | grep -oE '/static/js/index\.[0-9a-f]+\.js' \
    | head -n 1 || true
}

go_hash="$(extract_index 'https://go.wending.ai/')"
api_hash="$(extract_index 'https://api.wending.ai/')"

echo "go.wending.ai:  ${go_hash:-MISSING}"
echo "api.wending.ai: ${api_hash:-MISSING}"

if [[ -n "${go_hash}" && "${go_hash}" == "${api_hash}" ]]; then
  echo "OK: both frontends serve ${go_hash}"
  exit 0
fi

echo "DRIFT: frontends are out of sync (or a site failed to return an index hash)."
echo "go auto-deploys from Git when web/ changes (once the Vercel project is Git-linked);"
echo "the vultr-sv binary is a separate, explicit release. See docs/HANDOFF.md."
exit 1
