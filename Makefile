.PHONY: keys

KEYS_DIR := data/keys
SERVER_PRIVATE_KEY := $(KEYS_DIR)/server_private.pem
SERVER_PUBLIC_KEY := $(KEYS_DIR)/server_public.pem

keys:
	mkdir -p $(KEYS_DIR)
	openssl ecparam -name prime256v1 -genkey -noout -out $(SERVER_PRIVATE_KEY)
	openssl ec -in $(SERVER_PRIVATE_KEY) -pubout -out $(SERVER_PUBLIC_KEY)
