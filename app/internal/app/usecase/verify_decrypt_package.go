package usecase

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"electronic-digital-signature/internal/infra/encryption"
)

var (
	ErrInvalidEncryptedPackage = errors.New("invalid encrypted package")
	ErrInvalidSignature        = errors.New("invalid signature")
)

type decryptPackageVerifier interface {
	Verify(message []byte, signature []byte, publicKey []byte) error
}

type packageDecryptor interface {
	Decrypt(pkg encryption.EncryptedPackage) ([]byte, error)
}

type VerifyDecryptPackageInput struct {
	PackageContent []byte
}

type VerifyDecryptPackageMetadata struct {
	DocumentID          string
	Version             string
	EncryptionAlgorithm string
	KeyTransport        string
	SignatureAlgorithm  string
	OriginalFileName    string
	MimeType            string
	HashBase64          string
}

type VerifyDecryptPackageResult struct {
	Metadata          VerifyDecryptPackageMetadata
	DecryptedDocument []byte
}

type VerifyDecryptPackageUseCase struct {
	decryptor       packageDecryptor
	verifier        decryptPackageVerifier
	serverPublicKey []byte
}

func NewVerifyDecryptPackageUseCase(
	decryptor packageDecryptor,
	verifier decryptPackageVerifier,
	serverPublicKey []byte,
) *VerifyDecryptPackageUseCase {
	return &VerifyDecryptPackageUseCase{
		decryptor:       decryptor,
		verifier:        verifier,
		serverPublicKey: serverPublicKey,
	}
}

func (uc *VerifyDecryptPackageUseCase) Execute(ctx context.Context, input VerifyDecryptPackageInput) (*VerifyDecryptPackageResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if uc.decryptor == nil {
		return nil, fmt.Errorf("package decryptor is not configured")
	}
	if uc.verifier == nil {
		return nil, fmt.Errorf("signature verifier is not configured")
	}
	if len(uc.serverPublicKey) == 0 {
		return nil, fmt.Errorf("server public key is not configured")
	}
	if len(input.PackageContent) == 0 {
		return nil, fmt.Errorf("encrypted package is required")
	}

	pkg, err := encryption.DecodePackage(input.PackageContent)
	if err != nil {
		return nil, fmt.Errorf("%w", ErrInvalidEncryptedPackage)
	}

	result := &VerifyDecryptPackageResult{
		Metadata: metadataFromPackage(pkg),
	}

	decryptedDocument, err := uc.decryptor.Decrypt(pkg)
	if err != nil {
		return nil, fmt.Errorf("%w", ErrInvalidEncryptedPackage)
	}

	signature, err := base64.StdEncoding.DecodeString(pkg.SignatureBase64)
	if err != nil {
		return nil, fmt.Errorf("%w", ErrInvalidEncryptedPackage)
	}

	if err := uc.verifier.Verify(decryptedDocument, signature, uc.serverPublicKey); err != nil {
		return nil, fmt.Errorf("%w", ErrInvalidSignature)
	}

	result.DecryptedDocument = decryptedDocument
	return result, nil
}

func VerifyDecryptPackage(ctx context.Context, verifier decryptPackageVerifier, content, signature, publicKey []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return verifier.Verify(content, signature, publicKey)
}

func metadataFromPackage(pkg encryption.EncryptedPackage) VerifyDecryptPackageMetadata {
	return VerifyDecryptPackageMetadata{
		DocumentID:          pkg.DocumentID,
		Version:             pkg.Version,
		EncryptionAlgorithm: pkg.EncryptionAlgorithm,
		KeyTransport:        pkg.KeyTransport,
		SignatureAlgorithm:  pkg.SignatureAlgorithm,
		OriginalFileName:    pkg.OriginalFileName,
		MimeType:            pkg.MimeType,
		HashBase64:          pkg.HashBase64,
	}
}
