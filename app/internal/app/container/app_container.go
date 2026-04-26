package container

import (
	"electronic-digital-signature/internal/app/config"
	"electronic-digital-signature/internal/app/handler"
	"electronic-digital-signature/internal/app/usecase"
	"electronic-digital-signature/internal/domain/repository"
	infraauth "electronic-digital-signature/internal/infra/auth"
	"electronic-digital-signature/internal/infra/crypto"
	"electronic-digital-signature/internal/infra/database"
	"electronic-digital-signature/internal/infra/docx"
	"electronic-digital-signature/internal/infra/encryption"
	"electronic-digital-signature/internal/infra/id"
	"electronic-digital-signature/internal/infra/keys"
	"electronic-digital-signature/internal/infra/mailer"
	"electronic-digital-signature/internal/infra/storage"

	"gorm.io/gorm"
)

type AppContainer struct {
	ServerKeys         keys.ServerKeyPair
	DB                 *gorm.DB
	MessageRepository  *repository.MessageRepository
	DocumentRepository *repository.DocumentRepository
	UserRepository     *repository.UserRepository
	SignatureHandler   *handler.SignatureHandler
	DocumentHandler    *handler.DocumentHandler
	UserHandler        *handler.UserHandler
	AuthHandler        *handler.AuthHandler
	AuthMiddleware     *handler.AuthMiddleware
	Mailer             *mailer.SMTPMailer
}

func New(cfg config.Config) (*AppContainer, error) {
	serverKeys, err := keys.LoadServerKeyPair(
		cfg.ServerKeys.PrivateKeyPath,
		cfg.ServerKeys.PublicKeyPath,
		cfg.ServerKeys.PrivateKeyPEM,
		cfg.ServerKeys.PublicKeyPEM,
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
	userRepository := repository.NewUserRepository(db)
	smtpMailer := mailer.NewSMTPMailer(cfg.SMTP)
	documentStorage := storage.NewLocalDocumentStorage(cfg.DocumentStorage.Path)
	signatureProvider := crypto.NewECDSASHA256Provider()
	jwtManager := infraauth.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)
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
		documentStorage,
		idGenerator,
		docx.NewProcessor(),
		signatureProvider,
		serverKeys.PrivateKey,
	)
	sendDocumentUseCase := usecase.NewSendDocumentUseCase(
		documentRepository,
		documentStorage,
		signatureProvider,
		serverKeys.PrivateKey,
		encryption.NewDocumentEncryptor(documentStorage),
		smtpMailer,
	)
	verifyDecryptPackageUseCase := usecase.NewVerifyDecryptPackageUseCase(
		encryption.NewAESGCMEncryptor(),
		signatureProvider,
		serverKeys.PublicKey,
	)
	registerUserUseCase := usecase.NewRegisterUserUseCase(userRepository, idGenerator)
	getUserUseCase := usecase.NewGetUserUseCase(userRepository)
	loginUseCase := usecase.NewLoginUseCase(userRepository, jwtManager)
	currentUserUseCase := usecase.NewCurrentUserUseCase(userRepository)

	return &AppContainer{
		ServerKeys:         serverKeys,
		DB:                 db,
		MessageRepository:  messageRepository,
		DocumentRepository: documentRepository,
		UserRepository:     userRepository,
		Mailer:             smtpMailer,
		SignatureHandler: handler.NewSignatureHandler(
			serverKeys,
			verifyClientSignatureUseCase,
			issueServerSignedMessageUseCase,
			getServerSignedMessageUseCase,
		),
		DocumentHandler: handler.NewDocumentHandler(uploadDocumentUseCase, sendDocumentUseCase, verifyDecryptPackageUseCase),
		UserHandler:     handler.NewUserHandler(registerUserUseCase, getUserUseCase),
		AuthHandler:     handler.NewAuthHandler(loginUseCase, currentUserUseCase),
		AuthMiddleware:  handler.NewAuthMiddleware(jwtManager, currentUserUseCase),
	}, nil
}
