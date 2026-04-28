#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"
ACCESS_TOKEN="${ACCESS_TOKEN:?ACCESS_TOKEN is required}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_DIR="${SCRIPT_DIR}/out"
mkdir -p "${OUT_DIR}"

curl -s "${API_URL}/api/v1/server/public-key" > "${OUT_DIR}/server_public_key.json"
ruby -rjson -e '
  body = JSON.parse(File.read(ARGV[0]))
  File.write(ARGV[1], body.fetch("public_key_pem"))
' "${OUT_DIR}/server_public_key.json" "${OUT_DIR}/server_public.pem"

curl -s -X POST "${API_URL}/api/v1/server/messages" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d '{
    "message": "server signed message for scenario 2"
  }' > "${OUT_DIR}/server_message.json"

ruby -rjson -rbase64 -e '
  body = JSON.parse(File.read(ARGV[0]))
  File.write(ARGV[1], body.fetch("message"))
  File.binwrite(ARGV[2], Base64.decode64(body.fetch("signature_base64")))
' "${OUT_DIR}/server_message.json" "${OUT_DIR}/server_message.txt" "${OUT_DIR}/server_signature.bin"

openssl dgst -sha256 \
  -verify "${OUT_DIR}/server_public.pem" \
  -signature "${OUT_DIR}/server_signature.bin" \
  "${OUT_DIR}/server_message.txt"

echo "Saved files in ${OUT_DIR}"
