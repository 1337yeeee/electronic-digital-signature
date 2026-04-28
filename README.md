# Electronic Digital Signature Lab

## Quick demo

This README shows the current user-based flow of the lab:

1. generate server keys
2. register a user
3. login and get JWT
4. upload a document as that user
5. verify a user signature with the registered user key
6. request a server-signed message
7. send document package and inspect audit

## Run server

### 1. Generate server keys

```bash
make keys
```

This creates:

- `data/keys/server_private.pem`
- `data/keys/server_public.pem`

### 2. Prepare environment

```bash
cp .env.example .env
set -a
source .env
set +a
```

### 3. Start infrastructure

```bash
docker compose up -d postgres mailpit
```

Mailpit UI:

- `http://localhost:8025`

### 4. Run API

From the `app` directory:

```bash
cd app
go run ./cmd/server
```

By default:

- API: `http://localhost:8080`

### 5. Health check

```bash
curl http://localhost:8080/health
```

## OpenAPI

Client-facing API description lives in:

- `openapi.yaml`

It now includes:

- register/login/me
- JWT auth header
- document upload/send/audit
- user signature verification
- `401` and `403` errors

## Auth header

Protected endpoints require:

```http
Authorization: Bearer <access_token>
```

If the token is missing:

```json
{
  "success": false,
  "error": {
    "code": "unauthorized",
    "message": "Bearer token is required."
  }
}
```

If the token is invalid:

```json
{
  "success": false,
  "error": {
    "code": "invalid_token",
    "message": "Access token is invalid."
  }
}
```

If a document belongs to another user:

```json
{
  "success": false,
  "error": {
    "code": "forbidden",
    "message": "You do not have access to this document."
  }
}
```

## Curl scenarios

All commands below assume:

- server is running at `http://localhost:8080`
- environment is already loaded from `.env`
- commands run from repository root

### Register user

Generate user key pair:

```bash
mkdir -p data/user-keys
openssl ecparam -name prime256v1 -genkey -noout -out data/user-keys/user_private.pem
openssl ec -in data/user-keys/user_private.pem -pubout -out data/user-keys/user_public.pem
USER_PUBLIC_KEY_PEM="$(awk '{printf "%s\\n", $0}' data/user-keys/user_public.pem)"
```

Register:

```bash
curl -X POST http://localhost:8080/api/v1/users/register \
  -H 'Content-Type: application/json' \
  -d "{
    \"email\": \"user@example.com\",
    \"name\": \"Lab User\",
    \"password\": \"secret-password\",
    \"public_key_pem\": \"${USER_PUBLIC_KEY_PEM}\"
  }"
```

Example response:

```json
{
  "success": true,
  "data": {
    "id": "user-id",
    "email": "user@example.com",
    "name": "Lab User",
    "public_key_pem": "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----",
    "created_at": "2026-04-26T10:00:00Z",
    "updated_at": "2026-04-26T10:00:00Z"
  }
}
```

### Login

```bash
LOGIN_RESPONSE="$(
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "user@example.com",
    "password": "secret-password"
  }'
)"
echo "$LOGIN_RESPONSE"
```

Extract token:

```bash
ACCESS_TOKEN="$(printf '%s' "$LOGIN_RESPONSE" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')"
echo "$ACCESS_TOKEN"
```

### Get current user

```bash
curl http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

### Update current user public key

Generate a new public key and update it:

```bash
openssl ecparam -name prime256v1 -genkey -noout -out data/user-keys/user_private_v2.pem
openssl ec -in data/user-keys/user_private_v2.pem -pubout -out data/user-keys/user_public_v2.pem
USER_PUBLIC_KEY_V2_PEM="$(awk '{printf "%s\\n", $0}' data/user-keys/user_public_v2.pem)"

curl -X PUT http://localhost:8080/api/v1/users/me/public-key \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d "{
    \"public_key_pem\": \"${USER_PUBLIC_KEY_V2_PEM}\"
  }"
```

### Verify signature as current user

Prepare message and signature with the same private key whose public part is
registered in the account:

```bash
MESSAGE='user signed message'
printf '%s' "$MESSAGE" > /tmp/user-message.txt
openssl dgst -sha256 -sign data/user-keys/user_private_v2.pem -out /tmp/user-signature.bin /tmp/user-message.txt
USER_SIGNATURE_BASE64="$(base64 < /tmp/user-signature.bin | tr -d '\n')"
```

Verify by registered user key:

```bash
curl -X POST http://localhost:8080/api/v1/users/me/signatures/verify \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d "{
    \"message\": \"${MESSAGE}\",
    \"signature_base64\": \"${USER_SIGNATURE_BASE64}\"
  }"
```

Example response:

```json
{
  "valid": true,
  "signer_type": "user",
  "signer_user_id": "user-id"
}
```

### Get server public key

```bash
curl http://localhost:8080/api/v1/server/public-key
```

### Request server-signed message

```bash
curl -X POST http://localhost:8080/api/v1/server/messages \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d '{
    "message": "lab proof message"
  }'
```

Example response:

```json
{
  "message_id": "message-id",
  "signer_type": "server",
  "created_by_user_id": "user-id",
  "created_at": "2026-04-26T10:00:00Z",
  "message": "lab proof message",
  "algorithm": "ECDSA-SHA256",
  "hash_base64": "...",
  "signature_base64": "..."
}
```

### Upload document as user

The upload endpoint accepts only `.docx`.

```bash
curl -X POST http://localhost:8080/api/v1/documents \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -F 'recipient_email=recipient@example.com' \
  -F 'file=@./contract.docx;type=application/vnd.openxmlformats-officedocument.wordprocessingml.document'
```

Example response:

```json
{
  "success": true,
  "data": {
    "document_id": "document-id",
    "owner_user_id": "user-id",
    "signed_by_user_id": "user-id",
    "owner_email": "user@example.com",
    "recipient_email": "recipient@example.com",
    "original_file_name": "contract.docx",
    "stored_path": "data/uploads/document-id_contract.docx",
    "mime_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    "created_at": "2026-04-26T10:00:00Z"
  }
}
```

### Send document package

```bash
curl -X POST http://localhost:8080/api/v1/documents/document-id/send \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d '{
    "email": "recipient@example.com"
  }'
```

Example response:

```json
{
  "success": true,
  "data": {
    "document_id": "document-id",
    "owner_user_id": "user-id",
    "signed_by_user_id": "user-id",
    "sent_by_user_id": "user-id",
    "package_id": "document-id_encrypted_package",
    "recipient_email": "recipient@example.com",
    "send_status": "sent",
    "sent_at": "2026-04-26T10:01:00Z"
  }
}
```

Inspect outgoing email in Mailpit:

- `http://localhost:8025`

### Get document audit

```bash
curl http://localhost:8080/api/v1/documents/document-id/audit \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

Example response:

```json
{
  "success": true,
  "data": {
    "document_id": "document-id",
    "owner_user_id": "user-id",
    "signed_by_user_id": "user-id",
    "sent_by_user_id": "user-id",
    "owner_email": "user@example.com",
    "recipient_email": "recipient@example.com",
    "original_file_name": "contract.docx",
    "mime_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    "send_status": "sent",
    "created_at": "2026-04-26T10:00:00Z",
    "signed_at": "2026-04-26T10:00:00Z",
    "sent_at": "2026-04-26T10:01:00Z"
  }
}
```

### Verify and decrypt encrypted package

If you save the JSON attachment from Mailpit as `package.json`:

```bash
curl -X POST http://localhost:8080/api/v1/documents/verify-decrypt \
  -H 'Content-Type: application/json' \
  --data-binary @package.json
```

This endpoint is intentionally public for the lab demo.

### Legacy generic signature verification

The generic demo endpoint still exists if you want to verify a signature by
explicitly passing a PEM public key:

```bash
PUBLIC_KEY_PEM="$(awk '{printf "%s\\n", $0}' data/user-keys/user_public_v2.pem)"

curl -X POST http://localhost:8080/api/v1/signatures/verify \
  -H 'Content-Type: application/json' \
  -d "{
    \"message\": \"${MESSAGE}\",
    \"signature_base64\": \"${USER_SIGNATURE_BASE64}\",
    \"public_key\": \"${PUBLIC_KEY_PEM}\"
  }"
```

## Format notes

### Base64 fields

These fields use standard RFC 4648 base64:

- `signature_base64`
- `hash_base64`
- `encrypted_key_base64`
- `nonce_base64`
- `ciphertext_base64`
- `decrypted_document_base64`

### PEM public key

`public_key` and `public_key_pem` must be PEM-encoded ECDSA public keys, for
example:

```pem
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...
-----END PUBLIC KEY-----
```

## Current access policy

- `POST /api/v1/documents` requires authenticated user
- `POST /api/v1/documents/{id}/send` requires authenticated owner
- `GET /api/v1/documents/{id}/audit` requires authenticated owner
- `POST /api/v1/users/me/signatures/verify` requires authenticated user
- `POST /api/v1/server/messages` requires authenticated user
- `POST /api/v1/documents/verify-decrypt` is public for the lab demo

## Happy paths

There is also a ready-to-run demo kit for the three main laboratory scenarios:

- `demo/HAPPY_PATHS.md`

It contains:

1. Scenario 1: client signed, server verified
2. Scenario 2: server signed, client verified
3. Scenario 3: full document flow

### What the happy paths do

Scenario 1 demonstrates:

- prepared client key pair
- prepared signature of a fixed message
- verification through `POST /api/v1/signatures/verify`

Scenario 2 demonstrates:

- getting server public key
- requesting server-signed message
- verifying server signature locally with `openssl`

Scenario 3 demonstrates:

- uploading `.docx`
- server-side signing
- encrypted package creation
- sending through Mailpit
- downloading attachment from Mailpit
- verifying and decrypting package through API

### What is already prepared

Prepared demo artifacts are stored in `demo/`:

- `demo/scenario1/client_private.pem`
- `demo/scenario1/client_public.pem`
- `demo/scenario1/message.txt`
- `demo/scenario1/signature.base64`
- `demo/scenario3/contract.docx`

Prepared scripts:

- `demo/scenario1/verify_via_endpoint.sh`
- `demo/scenario2/request_and_verify.sh`
- `demo/scenario3/make_demo_docx.sh`
- `demo/scenario3/download_latest_mailpit_attachment.sh`
- `demo/scenario3/verify_decrypt.sh`

### What must be running

Before checking happy paths, you need:

1. exported environment from `.env`
2. running `postgres`
3. running `mailpit`
4. running API server

Minimal setup:

```bash
cp .env.example .env
set -a
source .env
set +a
docker compose up -d postgres mailpit
cd app
go run ./cmd/server
```

After that, API should be available at:

- `http://localhost:8080`

Mailpit UI:

- `http://localhost:8025`

### How to run

Open:

- `demo/HAPPY_PATHS.md`

That file contains:

- exact curl commands
- exact script commands
- expected outputs
- shared auth flow for scenarios 2 and 3

Quick start:

Scenario 1:

```bash
demo/scenario1/verify_via_endpoint.sh
```

Scenario 2:

1. register user
2. login and export `ACCESS_TOKEN`
3. run:

```bash
ACCESS_TOKEN="${ACCESS_TOKEN}" demo/scenario2/request_and_verify.sh
```

Scenario 3:

1. register user
2. login and export `ACCESS_TOKEN`
3. create or reuse demo document
4. upload and send document
5. download attachment from Mailpit
6. verify and decrypt package

Useful commands are already written in `demo/HAPPY_PATHS.md`.

### What to expect from each scenario

Scenario 1 expected result:

- API returns successful signature verification for the prepared client message

Scenario 2 expected result:

- server returns signed message
- local `openssl` verification prints `Verified OK`

Scenario 3 expected result:

- upload succeeds
- send succeeds
- Mailpit receives email with JSON attachment
- `verify-decrypt` returns `success: true` and `valid: true`

### Notes

- Scenario 2 and 3 require authenticated user flow.
- Scenario 3 uses Mailpit as dev mailer.
- `POST /api/v1/documents/verify-decrypt` is intentionally public for the lab demo.
- If you want the most up-to-date step list, use `demo/HAPPY_PATHS.md` as the primary source.
