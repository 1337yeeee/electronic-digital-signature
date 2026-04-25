package container

import (
	"electronic-digital-signature/internal/app/config"
	"electronic-digital-signature/internal/app/handler"
	"electronic-digital-signature/internal/app/usecase"
	"electronic-digital-signature/internal/domain/repository"
	"electronic-digital-signature/internal/infra/crypto"
	"electronic-digital-signature/internal/infra/database"
	"electronic-digital-signature/internal/infra/docx"
	"electronic-digital-signature/internal/infra/id"
	"electronic-digital-signature/internal/infra/keys"
	"electronic-digital-signature/internal/infra/storage"

	"gorm.io/gorm"
)

type AppContainer struct {
	ServerKeys         keys.ServerKeyPair
	DB                 *gorm.DB
	MessageRepository  *repository.MessageRepository
	DocumentRepository *repository.DocumentRepository
	SignatureHandler   *handler.SignatureHandler
	DocumentHandler    *handler.DocumentHandler
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
	documentRepository := repository.NewDocumentRepository(db)
	signatureProvider := crypto.NewECDSASHA256Provider()
	idGenerator := id.NewUUIDGenerator()
	verifyClientSignatureUseCase := usecase.NewVerifyClientSignatureUseCase(signatureProvider)
	issueServerSignedMessageUseCase := usecase.NewIssueServerSignedMessageUseCase(
		serverKeys.PrivateKey,
		signatureProvider,
		messageRepository,
		idGenerator,
		"server",
	)
	getServerSignedMessageUseCase := usecase.NewGetServerSignedMessageUseCase(messageRepository)
	uploadDocumentUseCase := usecase.NewUploadDocumentUseCase(
		documentRepository,
		storage.NewLocalDocumentStorage(cfg.DocumentStorage.Path),
		idGenerator,
		docx.NewProcessor(),
	)

	return &AppContainer{
		ServerKeys:         serverKeys,
		DB:                 db,
		MessageRepository:  messageRepository,
		DocumentRepository: documentRepository,
		SignatureHandler: handler.NewSignatureHandler(
			serverKeys,
			verifyClientSignatureUseCase,
			issueServerSignedMessageUseCase,
			getServerSignedMessageUseCase,
		),
		DocumentHandler: handler.NewDocumentHandler(uploadDocumentUseCase),
	}, nil
}
