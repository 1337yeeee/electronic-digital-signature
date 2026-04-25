package usecase

import "electronic-digital-signature/internal/domain/model"

type clientSignatureVerifier interface {
	Verify(message []byte, signature []byte, publicKey []byte) error
}

type VerifyClientSignatureUseCase struct {
	verifier clientSignatureVerifier
}

func NewVerifyClientSignatureUseCase(verifier clientSignatureVerifier) *VerifyClientSignatureUseCase {
	return &VerifyClientSignatureUseCase{
		verifier: verifier,
	}
}

func (uc *VerifyClientSignatureUseCase) Execute(message model.Message, signature []byte, publicKey []byte) error {
	return uc.verifier.Verify([]byte(message.Message), signature, publicKey)
}
