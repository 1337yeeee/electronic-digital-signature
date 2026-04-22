package usecase

import "context"

type decryptPackageVerifier interface {
	Verify(string, []byte, []byte) error
}

func VerifyDecryptPackage(ctx context.Context, verifier decryptPackageVerifier, content, signature, publicKey []byte) error {
	//TODO VerifyDecryptPackage
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return verifier.Verify(string(content), signature, publicKey)
}
