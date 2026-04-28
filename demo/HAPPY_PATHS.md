# Happy paths

This folder contains reproducible demonstration flows for all three lab
scenarios.

Default API URL:

```bash
export API_URL=http://localhost:8080
```

Default Mailpit URL:

```bash
export MAILPIT_URL=http://localhost:8025
```

## Before you start

From repository root:

```bash
cp .env.example .env
set -a
source .env
set +a
docker compose up -d postgres mailpit
cd app
go run ./cmd/server
```

Open Mailpit if needed:

- `http://localhost:8025`

## Scenario 1

Goal: client signed, server verified.

Prepared fixtures:

- `demo/scenario1/client_private.pem`
- `demo/scenario1/client_public.pem`
- `demo/scenario1/message.txt`
- `demo/scenario1/signature.base64`

Run:

```bash
demo/scenario1/verify_via_endpoint.sh
```

Raw curl:

```bash
MESSAGE="$(cat demo/scenario1/message.txt)"
SIGNATURE_BASE64="$(cat demo/scenario1/signature.base64)"
PUBLIC_KEY_PEM="$(awk '{printf "%s\\n", $0}' demo/scenario1/client_public.pem)"

curl -X POST "${API_URL}/api/v1/signatures/verify" \
  -H 'Content-Type: application/json' \
  -d "{
    \"message\": \"${MESSAGE}\",
    \"signature_base64\": \"${SIGNATURE_BASE64}\",
    \"public_key\": \"${PUBLIC_KEY_PEM}\"
  }"
```

Expected result:

```json
{
  "valid": true,
  "signer_type": "user"
}
```

## Common user flow for scenarios 2 and 3

Register demo user with the prepared client key:

```bash
USER_PUBLIC_KEY_PEM="$(awk '{printf "%s\\n", $0}' demo/scenario1/client_public.pem)"

curl -X POST "${API_URL}/api/v1/users/register" \
  -H 'Content-Type: application/json' \
  -d "{
    \"email\": \"demo-user@example.com\",
    \"name\": \"Demo User\",
    \"password\": \"secret-password\",
    \"public_key_pem\": \"${USER_PUBLIC_KEY_PEM}\"
  }"
```

Login:

```bash
LOGIN_RESPONSE="$(
curl -s -X POST "${API_URL}/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "demo-user@example.com",
    "password": "secret-password"
  }'
)"

ACCESS_TOKEN="$(printf '%s' "$LOGIN_RESPONSE" | ruby -rjson -e 'body = JSON.parse(STDIN.read); print body.dig("data", "access_token")')"
export ACCESS_TOKEN
echo "$ACCESS_TOKEN"
```

Check current user:

```bash
curl "${API_URL}/api/v1/auth/me" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

## Scenario 2

Goal: server signed, client verified.

Run:

```bash
ACCESS_TOKEN="${ACCESS_TOKEN}" demo/scenario2/request_and_verify.sh
```

Raw curl to get server public key:

```bash
curl "${API_URL}/api/v1/server/public-key"
```

Raw curl to request server-signed message:

```bash
curl -X POST "${API_URL}/api/v1/server/messages" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d '{
    "message": "server signed message for scenario 2"
  }'
```

What the script does:

1. downloads server public key
2. requests server-signed message
3. extracts `message` and `signature_base64`
4. verifies signature with `openssl dgst -sha256 -verify`

Output files:

- `demo/scenario2/out/server_public.pem`
- `demo/scenario2/out/server_message.json`
- `demo/scenario2/out/server_message.txt`
- `demo/scenario2/out/server_signature.bin`

## Scenario 3

Goal: full document flow.

Create demo DOCX:

```bash
demo/scenario3/make_demo_docx.sh
```

This creates:

- `demo/scenario3/contract.docx`

Upload document as authenticated user:

```bash
UPLOAD_RESPONSE="$(
curl -s -X POST "${API_URL}/api/v1/documents" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -F 'recipient_email=recipient@example.com' \
  -F 'file=@demo/scenario3/contract.docx;type=application/vnd.openxmlformats-officedocument.wordprocessingml.document'
)"

echo "$UPLOAD_RESPONSE"
DOCUMENT_ID="$(printf '%s' "$UPLOAD_RESPONSE" | ruby -rjson -e 'body = JSON.parse(STDIN.read); print body.dig("data", "document_id")')"
export DOCUMENT_ID
echo "$DOCUMENT_ID"
```

Send encrypted package by dev-mailer:

```bash
curl -X POST "${API_URL}/api/v1/documents/${DOCUMENT_ID}/send" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d '{
    "email": "recipient@example.com"
  }'
```

Inspect audit:

```bash
curl "${API_URL}/api/v1/documents/${DOCUMENT_ID}/audit" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

Download latest JSON attachment from Mailpit:

```bash
demo/scenario3/download_latest_mailpit_attachment.sh
```

This saves:

- `demo/scenario3/package.json`

Verify and decrypt package:

```bash
demo/scenario3/verify_decrypt.sh
```

Raw curl for verify/decrypt:

```bash
curl -X POST "${API_URL}/api/v1/documents/verify-decrypt" \
  -H 'Content-Type: application/json' \
  --data-binary @demo/scenario3/package.json
```

Expected result:

```json
{
  "success": true,
  "data": {
    "valid": true,
    "metadata": {
      "document_id": "..."
    },
    "decrypted_document_base64": "..."
  }
}
```
