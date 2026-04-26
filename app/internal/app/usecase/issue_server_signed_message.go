package usecase

import (
	"context"
	"fmt"
	"time"

	"electronic-digital-signature/internal/domain/model"
)

type serverMessageSigner interface {
	Hash(message []byte) []byte
	Sign(message []byte, privateKey []byte) ([]byte, error)
}

type serverMessageRepository interface {
	Create(ctx context.Context, message *model.Message) error
}

type messageIDGenerator interface {
	Generate() (string, error)
}

type IssueServerSignedMessageUseCase struct {
	privateKey  []byte
	signer      serverMessageSigner
	repository  serverMessageRepository
	idGenerator messageIDGenerator
	signerID    string
}

func NewIssueServerSignedMessageUseCase(
	privateKey []byte,
	signer serverMessageSigner,
	repository serverMessageRepository,
	idGenerator messageIDGenerator,
	signerID string,
) *IssueServerSignedMessageUseCase {
	return &IssueServerSignedMessageUseCase{
		privateKey:  privateKey,
		signer:      signer,
		repository:  repository,
		idGenerator: idGenerator,
		signerID:    signerID,
	}
}

func (uc *IssueServerSignedMessageUseCase) Execute(ctx context.Context, message *model.Message) (signature []byte, messageHash []byte, err error) {
	if uc.repository == nil {
		return nil, nil, fmt.Errorf("message repository is not configured")
	}
	if uc.idGenerator == nil {
		return nil, nil, fmt.Errorf("message id generator is not configured")
	}

	if message.ID == "" {
		message.ID, err = uc.idGenerator.Generate()
		if err != nil {
			return nil, nil, err
		}
	}
	if message.SignerID == "" {
		message.SignerID = uc.signerID
	}
	if message.CreatedByUserID == "" {
		return nil, nil, fmt.Errorf("created by user id is required")
	}
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now().UTC()
	}
	message.VerificationStatus = model.VerificationStatusPending

	byteMessage := []byte(message.Message)
	messageHash = uc.signer.Hash(byteMessage)
	signature, err = uc.signer.Sign(byteMessage, uc.privateKey)
	if err != nil {
		return nil, messageHash, err
	}

	message.Hash = messageHash
	message.Signature = signature
	message.VerificationStatus = model.VerificationStatusValid
	message.SignedAt = time.Now().UTC()

	if err := uc.repository.Create(ctx, message); err != nil {
		return nil, nil, fmt.Errorf("save signed message: %w", err)
	}

	return signature, messageHash, nil
}
