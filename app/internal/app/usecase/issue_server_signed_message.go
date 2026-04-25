package usecase

import "electronic-digital-signature/internal/domain/model"

type serverMessageSigner interface {
	Hash(message []byte) []byte
	Sign(message []byte, privateKey []byte) ([]byte, error)
}

type IssueServerSignedMessageUseCase struct {
	privateKey []byte
	signer     serverMessageSigner
}

func NewIssueServerSignedMessageUseCase(privateKey []byte, signer serverMessageSigner) *IssueServerSignedMessageUseCase {
	return &IssueServerSignedMessageUseCase{
		privateKey: privateKey,
		signer:     signer,
	}
}

func (uc *IssueServerSignedMessageUseCase) Execute(message *model.Message) (signature []byte, messageHash []byte, err error) {
	byteMessage := []byte(message.Message)
	messageHash = uc.signer.Hash(byteMessage)
	signature, err = uc.signer.Sign(byteMessage, uc.privateKey)
	return
}
