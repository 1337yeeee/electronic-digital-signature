package keys

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type ServerKeyPair struct {
	PrivateKey []byte
	PublicKey  []byte
}

func LoadServerKeyPair(privateKeyPath, publicKeyPath, privateKeyPEM, publicKeyPEM string) (ServerKeyPair, error) {
	privateKey, err := readRequiredKey("server private key", privateKeyPath, privateKeyPEM)
	if err != nil {
		return ServerKeyPair{}, err
	}

	publicKey, err := readRequiredKey("server public key", publicKeyPath, publicKeyPEM)
	if err != nil {
		return ServerKeyPair{}, err
	}

	return ServerKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

func readRequiredKey(name, path, raw string) ([]byte, error) {
	if strings.TrimSpace(raw) != "" {
		return validateRequiredKeyBytes(name, []byte(raw), "environment")
	}

	return readRequiredKeyFile(name, path)
}

func readRequiredKeyFile(name, path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("%s path is empty", name)
	}

	key, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s file does not exist: %s", name, path)
		}

		return nil, fmt.Errorf("read %s file %q: %w", name, path, err)
	}

	return validateRequiredKeyBytes(name, key, path)
}

func validateRequiredKeyBytes(name string, key []byte, source string) ([]byte, error) {
	if len(strings.TrimSpace(string(key))) == 0 {
		return nil, fmt.Errorf("%s file is empty: %s", name, source)
	}

	return key, nil
}
