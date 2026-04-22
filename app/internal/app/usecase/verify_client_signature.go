package usecase

import "electronic-digital-signature/internal/domain/model"

type clientSignatureVerifier interface {
	Verify(message []byte, signature []byte, publicKey []byte) error
}

func VerifyClientSignature(message model.Message, signature []byte, publicKey []byte, provider clientSignatureVerifier) error {
	//TODO VerifyClientSignature
	return provider.Verify([]byte(message.Message), signature, publicKey)
}
