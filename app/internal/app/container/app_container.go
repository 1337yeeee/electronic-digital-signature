package container

import (
	"electronic-digital-signature/internal/app/config"
	"electronic-digital-signature/internal/infra/keys"
)

type AppContainer struct {
	ServerKeys keys.ServerKeyPair
}

func New(cfg config.Config) (*AppContainer, error) {
	serverKeys, err := keys.LoadServerKeyPair(
		cfg.ServerKeys.PrivateKeyPath,
		cfg.ServerKeys.PublicKeyPath,
	)
	if err != nil {
		return nil, err
	}

	return &AppContainer{
		ServerKeys: serverKeys,
	}, nil
}
