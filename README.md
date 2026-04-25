# Electronic Digital Signature Lab

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
```
