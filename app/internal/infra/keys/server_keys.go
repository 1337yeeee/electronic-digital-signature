package keys

import (
	"errors"
	"fmt"
	"os"
)

type ServerKeyPair struct {
	PrivateKey []byte
	PublicKey  []byte
}

func LoadServerKeyPair(privateKeyPath, publicKeyPath string) (ServerKeyPair, error) {
	privateKey, err := readRequiredKeyFile("server private key", privateKeyPath)
	if err != nil {
		return ServerKeyPair{}, err
	}

	publicKey, err := readRequiredKeyFile("server public key", publicKeyPath)
	if err != nil {
		return ServerKeyPair{}, err
	}

	return ServerKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
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

	if len(key) == 0 {
		return nil, fmt.Errorf("%s file is empty: %s", name, path)
	}

	return key, nil
}
