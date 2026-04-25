package container

import (
	"electronic-digital-signature/internal/app/config"
	"electronic-digital-signature/internal/app/handler"
	"electronic-digital-signature/internal/infra/crypto"
	"electronic-digital-signature/internal/infra/keys"
)

type AppContainer struct {
	ServerKeys       keys.ServerKeyPair
	SignatureHandler *handler.SignatureHandler
}

func New(cfg config.Config) (*AppContainer, error) {
	serverKeys, err := keys.LoadServerKeyPair(
		cfg.ServerKeys.PrivateKeyPath,
		cfg.ServerKeys.PublicKeyPath,
	)
	if err != nil {
		return nil, err
	}

	signatureProvider := crypto.NewECDSASHA256Provider()

	return &AppContainer{
		ServerKeys:       serverKeys,
		SignatureHandler: handler.NewSignatureHandler(serverKeys, signatureProvider),
	}, nil
}
