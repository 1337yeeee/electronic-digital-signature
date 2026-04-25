package usecase

import "context"

type decryptPackageVerifier interface {
	Verify(message []byte, signature []byte, publicKey []byte) error
}

func VerifyDecryptPackage(ctx context.Context, verifier decryptPackageVerifier, content, signature, publicKey []byte) error {
	//TODO VerifyDecryptPackage
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return verifier.Verify(content, signature, publicKey)
}
