package usecase

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"electronic-digital-signature/internal/domain/model"
	"electronic-digital-signature/internal/infra/encryption"
)

const (
	DocumentSendStatusSent   = "sent"
	DocumentSendStatusFailed = "failed"
)

var ErrDocumentAccessDenied = errors.New("document access denied")

type EmailAttachment struct {
	FileName    string
	ContentType string
	Content     []byte
}

type Mailer interface {
	SendEmail(ctx context.Context, to []string, subject, body string, attachments []EmailAttachment) error
}

type sendDocumentRepository interface {
	FindByID(ctx context.Context, id string) (*model.Document, error)
	Update(ctx context.Context, document *model.Document) error
}

type secureDocumentStorage interface {
	Read(ctx context.Context, path string) ([]byte, error)
}

type DocumentSigner interface {
	Hash(message []byte) []byte
	Sign(message []byte, privateKey []byte) ([]byte, error)
}

type DocumentEncryptor interface {
	EncryptAndSave(ctx context.Context, document model.Document, content []byte) (encryption.EncryptedPackage, string, error)
}

type SendSecureDocumentInput struct {
	Document         model.Document
	To               []string
	Subject          string
	EncryptedPackage []byte
	AttachmentName   string
}

type SendDocumentInput struct {
	DocumentID     string
	RecipientEmail string
	SentByUserID   string
}

type SendDocumentResult struct {
	DocumentID     string
	OwnerUserID    string
	SignedByUserID string
	PackageID      string
	RecipientEmail string
	SendStatus     string
	SentByUserID   string
	SentAt         *time.Time
}

type SendDocumentUseCase struct {
	repository sendDocumentRepository
	storage    secureDocumentStorage
	signer     DocumentSigner
	privateKey []byte
	encryptor  DocumentEncryptor
	mailer     Mailer
}

func NewSendDocumentUseCase(
	repository sendDocumentRepository,
	storage secureDocumentStorage,
	signer DocumentSigner,
	privateKey []byte,
	encryptor DocumentEncryptor,
	mailer Mailer,
) *SendDocumentUseCase {
	return &SendDocumentUseCase{
		repository: repository,
		storage:    storage,
		signer:     signer,
		privateKey: privateKey,
		encryptor:  encryptor,
		mailer:     mailer,
	}
}

func (uc *SendDocumentUseCase) Execute(ctx context.Context, input SendDocumentInput) (*SendDocumentResult, error) {
	if uc.repository == nil {
		return nil, fmt.Errorf("document repository is not configured")
	}
	if uc.storage == nil {
		return nil, fmt.Errorf("document storage is not configured")
	}
	if uc.mailer == nil {
		return nil, fmt.Errorf("mailer is not configured")
	}
	if strings.TrimSpace(input.DocumentID) == "" {
		return nil, fmt.Errorf("document_id is required")
	}
	if strings.TrimSpace(input.RecipientEmail) == "" {
		return nil, fmt.Errorf("recipient email is required")
	}
	if strings.TrimSpace(input.SentByUserID) == "" {
		return nil, fmt.Errorf("sent by user id is required")
	}

	document, err := uc.repository.FindByID(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(document.OwnerUserID) != "" && document.OwnerUserID != input.SentByUserID {
		return nil, ErrDocumentAccessDenied
	}

	encryptedPackage, attachmentName, packageID, err := uc.loadOrCreateEncryptedPackage(ctx, document)
	if err != nil {
		return nil, err
	}

	subject := fmt.Sprintf("Encrypted document package: %s", document.OriginalFileName)
	sendErr := SendSecureDocument(ctx, uc.mailer, SendSecureDocumentInput{
		Document:         *document,
		To:               []string{input.RecipientEmail},
		Subject:          subject,
		EncryptedPackage: encryptedPackage,
		AttachmentName:   attachmentName,
	})

	now := time.Now().UTC()
	document.RecipientEmail = input.RecipientEmail
	document.LastSentByUserID = input.SentByUserID
	document.LastSentToEmail = input.RecipientEmail
	document.SendError = ""
	if sendErr != nil {
		document.SendStatus = DocumentSendStatusFailed
		document.SendError = sendErr.Error()
		document.SentAt = nil
	} else {
		document.SendStatus = DocumentSendStatusSent
		document.SentAt = &now
	}

	if err := uc.repository.Update(ctx, document); err != nil {
		return nil, fmt.Errorf("save document send status: %w", err)
	}
	if sendErr != nil {
		return nil, sendErr
	}

	return &SendDocumentResult{
		DocumentID:     document.ID,
		OwnerUserID:    document.OwnerUserID,
		SignedByUserID: document.SignedByUserID,
		PackageID:      packageID,
		RecipientEmail: input.RecipientEmail,
		SendStatus:     document.SendStatus,
		SentByUserID:   document.LastSentByUserID,
		SentAt:         document.SentAt,
	}, nil
}

func (uc *SendDocumentUseCase) loadOrCreateEncryptedPackage(ctx context.Context, document *model.Document) ([]byte, string, string, error) {
	if document.EncryptedPath != "" {
		content, err := uc.storage.Read(ctx, document.EncryptedPath)
		if err != nil {
			return nil, "", "", fmt.Errorf("read encrypted package: %w", err)
		}

		attachmentName := filepath.Base(document.EncryptedPath)
		return content, attachmentName, packageIDFromAttachmentName(attachmentName), nil
	}

	if uc.signer == nil {
		return nil, "", "", fmt.Errorf("document signer is not configured")
	}
	if len(uc.privateKey) == 0 {
		return nil, "", "", fmt.Errorf("server private key is not configured")
	}

	if uc.encryptor == nil {
		return nil, "", "", fmt.Errorf("document encryptor is not configured")
	}

	content, err := uc.storage.Read(ctx, document.StoredPath)
	if err != nil {
		return nil, "", "", fmt.Errorf("read stored document: %w", err)
	}

	document.Hash = uc.signer.Hash(content)
	signature, err := uc.signer.Sign(content, uc.privateKey)
	if err != nil {
		return nil, "", "", fmt.Errorf("sign document package content: %w", err)
	}
	now := time.Now().UTC()
	document.Signature = signature
	document.SignedAt = now

	_, encryptedPath, err := uc.encryptor.EncryptAndSave(ctx, *document, content)
	if err != nil {
		return nil, "", "", fmt.Errorf("create encrypted package: %w", err)
	}
	document.EncryptedPath = encryptedPath

	encryptedPackage, err := uc.storage.Read(ctx, encryptedPath)
	if err != nil {
		return nil, "", "", fmt.Errorf("read encrypted package: %w", err)
	}

	attachmentName := filepath.Base(encryptedPath)
	return encryptedPackage, attachmentName, packageIDFromAttachmentName(attachmentName), nil
}

func SendSecureDocument(ctx context.Context, mailer Mailer, input SendSecureDocumentInput) error {
	if len(input.EncryptedPackage) == 0 {
		return fmt.Errorf("encrypted package is required")
	}

	attachmentName := input.AttachmentName
	if attachmentName == "" {
		attachmentName = input.Document.ID + "_encrypted_package.json"
	}

	body := fmt.Sprintf(
		"Encrypted document package is attached.\n\nDocument ID: %s\nEncryption algorithm: %s\nKey transport: %s\nSignature algorithm: %s\n\nUse the package fields nonce_base64, ciphertext_base64 and encrypted_key_base64 to decrypt the document with %s, then verify signature_base64 against hash_base64 with %s.",
		input.Document.ID,
		encryption.AESGCMAlgorithm,
		encryption.PlaintextDemoKey,
		encryption.SignatureAlgorithm,
		encryption.AESGCMAlgorithm,
		encryption.SignatureAlgorithm,
	)

	return mailer.SendEmail(ctx, input.To, input.Subject, body, []EmailAttachment{
		{
			FileName:    attachmentName,
			ContentType: "application/json",
			Content:     input.EncryptedPackage,
		},
	})
}

func packageIDFromAttachmentName(attachmentName string) string {
	return strings.TrimSuffix(attachmentName, filepath.Ext(attachmentName))
}
