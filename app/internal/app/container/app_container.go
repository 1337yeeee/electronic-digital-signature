package container

import (
	"electronic-digital-signature/internal/app/config"
	"electronic-digital-signature/internal/app/handler"
	"electronic-digital-signature/internal/app/usecase"
	"electronic-digital-signature/internal/domain/repository"
	"electronic-digital-signature/internal/infra/crypto"
	"electronic-digital-signature/internal/infra/database"
	"electronic-digital-signature/internal/infra/id"
	"electronic-digital-signature/internal/infra/keys"

	"gorm.io/gorm"
)

type AppContainer struct {
	ServerKeys        keys.ServerKeyPair
	DB                *gorm.DB
	MessageRepository *repository.MessageRepository
	SignatureHandler  *handler.SignatureHandler
}

func New(cfg config.Config) (*AppContainer, error) {
	serverKeys, err := keys.LoadServerKeyPair(
		cfg.ServerKeys.PrivateKeyPath,
		cfg.ServerKeys.PublicKeyPath,
	)
	if err != nil {
		return nil, err
	}

	db, err := database.NewPostgres(cfg)
	if err != nil {
		return nil, err
	}
	if err := database.AutoMigrate(db); err != nil {
		return nil, err
	}

	messageRepository := repository.NewMessageRepository(db)
	signatureProvider := crypto.NewECDSASHA256Provider()
	verifyClientSignatureUseCase := usecase.NewVerifyClientSignatureUseCase(signatureProvider)
	issueServerSignedMessageUseCase := usecase.NewIssueServerSignedMessageUseCase(
		serverKeys.PrivateKey,
		signatureProvider,
		messageRepository,
		id.NewUUIDGenerator(),
		"server",
	)

	return &AppContainer{
		ServerKeys:        serverKeys,
		DB:                db,
		MessageRepository: messageRepository,
		SignatureHandler: handler.NewSignatureHandler(
			serverKeys,
			verifyClientSignatureUseCase,
			issueServerSignedMessageUseCase,
		),
	}, nil
}
