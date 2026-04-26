# Electronic Digital Signature Lab

## Quick demo

This section is a short end-to-end сценарий for lab demonstration.

### 1. Generate server keys

```bash
make keys
```

This creates:

- `data/keys/server_private.pem`
- `data/keys/server_public.pem`

### 2. Prepare environment

Copy example config:

```bash
cp .env.example .env
```

Important: the Go server does not load `.env` automatically. Before running the
server, export variables into the shell:

```bash
set -a
source .env
set +a
```

### 3. Start infrastructure

```bash
docker compose up -d postgres mailpit
```

Mailpit UI for email inspection:

- `http://localhost:8025`

### 4. Run server

From the `app` directory:

```bash
cd app
go run ./cmd/server
```

By default, the server listens on:

- `http://localhost:8080`

### 5. Check health

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{
  "success": true,
  "data": {
    "status": "ok"
  }
}
```

## API specification

The project includes a client-facing OpenAPI description in
`openapi.yaml` at the repository root.

You can use it to:

- review all endpoints, requests, responses, and error formats;
- inspect base64 and PEM field formats;
- import the API into Swagger UI, Redoc, or client generators.

## Environment

Copy `.env.example` to `.env` and adjust values if needed.

The current server uses only these variable groups:

- API: `API_PORT`
- Postgres: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `SSL_MODE`
- Server keys: `SERVER_PRIVATE_KEY_PATH`, `SERVER_PUBLIC_KEY_PATH`, `SERVER_PRIVATE_KEY_PEM`, `SERVER_PUBLIC_KEY_PEM`
- Storage: `DOCUMENT_STORAGE_PATH`
- SMTP: `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`, `SMTP_USER`, `SMTP_PASSWORD`
- Docker convenience: `POSTGRES_CONTAINER_NAME`, `MAILPIT_CONTAINER_NAME`, `MAILPIT_UI_PORT`

Server keys can be provided either by file path:

```bash
export SERVER_PRIVATE_KEY_PATH=data/keys/server_private.pem
export SERVER_PUBLIC_KEY_PATH=data/keys/server_public.pem
```

or directly as PEM values:

```bash
export SERVER_PRIVATE_KEY_PEM='-----BEGIN EC PRIVATE KEY-----
...
-----END EC PRIVATE KEY-----'
export SERVER_PUBLIC_KEY_PEM='-----BEGIN PUBLIC KEY-----
...
-----END PUBLIC KEY-----'
```

## Curl scenarios

All examples below assume:

- server is running at `http://localhost:8080`
- current shell already loaded `.env`
- commands are run from repository root unless noted

### Get server public key

```bash
curl http://localhost:8080/api/v1/server/public-key
```

Example response:

```json
{
  "algorithm": "ECDSA-SHA256",
  "public_key_pem": "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----\n"
}
```

### Request server-signed message

With custom message:

```bash
curl -X POST http://localhost:8080/api/v1/server/messages \
  -H 'Content-Type: application/json' \
  -d '{
    "message": "lab proof message"
  }'
```

Without custom message, server generates one:

```bash
curl -X POST http://localhost:8080/api/v1/server/messages
```

Example response:

```json
{
  "message_id": "00000000-0000-4000-8000-000000000001",
  "created_at": "2026-04-26T10:00:00Z",
  "message": "lab proof message",
  "algorithm": "ECDSA-SHA256",
  "hash_base64": "...",
  "signature_base64": "..."
}
```

### Verify client signature

Generate client key pair:

```bash
mkdir -p data/client-keys
openssl ecparam -name prime256v1 -genkey -noout -out data/client-keys/client_private.pem
openssl ec -in data/client-keys/client_private.pem -pubout -out data/client-keys/client_public.pem
```

Prepare a message and signature:

```bash
MESSAGE='client signed message'
printf '%s' "$MESSAGE" > /tmp/client-message.txt
openssl dgst -sha256 -sign data/client-keys/client_private.pem -out /tmp/client-signature.bin /tmp/client-message.txt
SIGNATURE_BASE64="$(base64 < /tmp/client-signature.bin | tr -d '\n')"
PUBLIC_KEY_PEM="$(awk '{printf "%s\\n", $0}' data/client-keys/client_public.pem)"
```

Verify signature:

```bash
curl -X POST http://localhost:8080/api/v1/signatures/verify \
  -H 'Content-Type: application/json' \
  -d "{
    \"message\": \"${MESSAGE}\",
    \"signature_base64\": \"${SIGNATURE_BASE64}\",
    \"public_key\": \"${PUBLIC_KEY_PEM}\"
  }"
```

Valid response:

```json
{
  "valid": true
}
```

Invalid response example:

```json
{
  "valid": false,
  "error": "invalid signature"
}
```

### Upload and send document

The upload endpoint accepts only `.docx` files.

Upload:

```bash
curl -X POST http://localhost:8080/api/v1/documents \
  -F 'owner_email=owner@example.com' \
  -F 'recipient_email=recipient@example.com' \
  -F 'file=@./contract.docx;type=application/vnd.openxmlformats-officedocument.wordprocessingml.document'
```

Example response:

```json
{
  "success": true,
  "data": {
    "document_id": "00000000-0000-4000-8000-000000000001",
    "owner_email": "owner@example.com",
    "recipient_email": "recipient@example.com",
    "original_file_name": "contract.docx",
    "stored_path": "data/uploads/00000000-0000-4000-8000-000000000001_contract.docx",
    "mime_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    "created_at": "2026-04-26T10:00:00Z"
  }
}
```

Send encrypted package by email:

```bash
curl -X POST http://localhost:8080/api/v1/documents/00000000-0000-4000-8000-000000000001/send \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "recipient@example.com"
  }'
```

Example response:

```json
{
  "success": true,
  "data": {
    "document_id": "00000000-0000-4000-8000-000000000001",
    "package_id": "00000000-0000-4000-8000-000000000001_encrypted_package",
    "recipient_email": "recipient@example.com",
    "send_status": "sent",
    "sent_at": "2026-04-26T10:01:00Z"
  }
}
```

Inspect the outgoing email and attachment in Mailpit:

```text
http://localhost:8025
```

### Verify and decrypt encrypted package

If you save the JSON attachment from Mailpit as `package.json`, you can verify
and decrypt it like this:

```bash
curl -X POST http://localhost:8080/api/v1/documents/verify-decrypt \
  -H 'Content-Type: application/json' \
  --data-binary @package.json
```

Example response:

```json
{
  "success": true,
  "data": {
    "valid": true,
    "metadata": {
      "document_id": "00000000-0000-4000-8000-000000000001",
      "version": "1",
      "encryption_algorithm": "AES-256-GCM",
      "key_transport": "plaintext_demo",
      "signature_algorithm": "ECDSA-SHA256",
      "original_file_name": "contract.docx",
      "mime_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
      "hash_base64": "..."
    },
    "decrypted_document_base64": "..."
  }
}
```

Corrupted package example:

```json
{
  "success": false,
  "error": {
    "code": "invalid_package",
    "message": "Encrypted package is invalid."
  }
}
```

## Encrypted document package

Documents are encrypted with AES-256-GCM. The current demo format uses a
single-use random AES key and stores that key in the package as base64 with
`key_transport: "plaintext_demo"`.

This is only for the laboratory demo while recipient public-key encryption is
not implemented yet. When recipient encryption is added, the AES key should be
encrypted with the recipient public key and `key_transport` should change.

Package JSON fields:

```json
{
  "version": "1",
  "document_id": "...",
  "encryption_algorithm": "AES-256-GCM",
  "key_transport": "plaintext_demo",
  "encrypted_key_base64": "...",
  "nonce_base64": "...",
  "ciphertext_base64": "...",
  "signature_base64": "...",
  "hash_base64": "...",
  "signature_algorithm": "ECDSA-SHA256",
  "original_file_name": "document.docx",
  "mime_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
}
```

When saved locally, the encrypted package file is named
`<document_id>_encrypted_package.json`.
