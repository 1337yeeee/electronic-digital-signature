#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_FILE="${SCRIPT_DIR}/package.json"

test -f "${PACKAGE_FILE}" || {
  echo "Missing ${PACKAGE_FILE}. Download attachment first." >&2
  exit 1
}

curl -X POST "${API_URL}/api/v1/documents/verify-decrypt" \
  -H 'Content-Type: application/json' \
  --data-binary @"${PACKAGE_FILE}"
echo
