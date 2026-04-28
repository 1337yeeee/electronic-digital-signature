#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

MESSAGE="$(cat "${SCRIPT_DIR}/message.txt")"
SIGNATURE_BASE64="$(cat "${SCRIPT_DIR}/signature.base64")"
PUBLIC_KEY_PEM="$(awk '{printf "%s\\n", $0}' "${SCRIPT_DIR}/client_public.pem")"

curl -X POST "${API_URL}/api/v1/signatures/verify" \
  -H 'Content-Type: application/json' \
  -d "{
    \"message\": \"${MESSAGE}\",
    \"signature_base64\": \"${SIGNATURE_BASE64}\",
    \"public_key\": \"${PUBLIC_KEY_PEM}\"
  }"
echo
