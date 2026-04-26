# Electronic Digital Signature Lab

## API specification

The project includes a client-facing OpenAPI description in
`openapi.yaml` at the repository root.

You can use it to:

- review all endpoints, requests, responses, and error formats;
- inspect base64 and PEM field formats;
- import the API into Swagger UI, Redoc, or client generators.

## Server keys

The server expects an ECDSA private/public key pair in PEM format. For local
development, generate the keys with Make:

```bash
make keys
```

The command runs these OpenSSL calls:

```bash
mkdir -p data/keys
openssl ecparam -name prime256v1 -genkey -noout -out data/keys/server_private.pem
openssl ec -in data/keys/server_private.pem -pubout -out data/keys/server_public.pem
```

Point the application to these files with environment variables:

```bash
export SERVER_PRIVATE_KEY_PATH=data/keys/server_private.pem
export SERVER_PUBLIC_KEY_PATH=data/keys/server_public.pem
```

Keep private keys out of git. Files matching `data/keys/*.pem` are ignored by
default.

## Local Postgres

Start PostgreSQL for local development:

```bash
docker compose up -d postgres
```

Use these environment variables for the application:

```bash
export API_PORT=8080
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=eds_lab
export SSL_MODE=disable
export SMTP_HOST=localhost
export SMTP_PORT=1025
export SMTP_FROM=server@example.com
```

Mailpit is available at http://localhost:8025 for local email inspection.

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
