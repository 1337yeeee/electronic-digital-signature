package usecase

import "electronic-digital-signature/internal/domain/model"

type serverMessageSigner interface {
	Hash(message []byte) []byte
	Sign(message []byte, privateKey []byte) ([]byte, error)
}

func IssueServerSignedMessage(
	//TODO IssueServerSignedMessage
	message *model.Message,
	privateKey []byte,
	provider serverMessageSigner,
) (signature []byte, messageHash []byte, err error) {
	byteMessage := []byte(message.Message)
	messageHash = provider.Hash(byteMessage)
	signature, err = provider.Sign(byteMessage, privateKey)
	return
}
