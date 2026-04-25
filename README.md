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
