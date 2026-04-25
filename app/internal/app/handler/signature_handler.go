package handler

import (
	"encoding/base64"
	"net/http"

	"electronic-digital-signature/internal/app/dto"
	"electronic-digital-signature/internal/app/usecase"
	"electronic-digital-signature/internal/domain/model"
	"electronic-digital-signature/internal/infra/crypto"
	"electronic-digital-signature/internal/infra/keys"

	"github.com/gin-gonic/gin"
)

type signatureVerifier interface {
	Verify(message []byte, signature []byte, publicKey []byte) error
}

type SignatureHandler struct {
	serverKeys keys.ServerKeyPair
	verifier   signatureVerifier
}

func NewSignatureHandler(serverKeys keys.ServerKeyPair, verifier signatureVerifier) *SignatureHandler {
	return &SignatureHandler{
		serverKeys: serverKeys,
		verifier:   verifier,
	}
}

func (h *SignatureHandler) GetServerPublicKey(ctx *gin.Context) {
	if len(h.serverKeys.PublicKey) == 0 {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "server public key is not loaded",
		})
		return
	}

	ctx.JSON(http.StatusOK, dto.ServerPublicKeyResponse{
		Algorithm:    crypto.ECDSASHA256Algorithm,
		PublicKeyPEM: string(h.serverKeys.PublicKey),
	})
}

func (h *SignatureHandler) VerifyClientSignature(ctx *gin.Context) {
	if h.verifier == nil {
		ctx.JSON(http.StatusInternalServerError, dto.VerifyClientSignatureResponse{
			Valid: false,
			Error: "signature verifier is not configured",
		})
		return
	}

	var request dto.VerifyClientSignatureRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.VerifyClientSignatureResponse{
			Valid: false,
			Error: "invalid request body",
		})
		return
	}

	signature, err := base64.StdEncoding.DecodeString(request.SignatureBase64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.VerifyClientSignatureResponse{
			Valid: false,
			Error: "signature_base64 must be valid base64",
		})
		return
	}

	message := model.Message{Message: request.Message}
	if err := usecase.VerifyClientSignature(message, signature, []byte(request.PublicKey), h.verifier); err != nil {
		ctx.JSON(http.StatusOK, dto.VerifyClientSignatureResponse{
			Valid: false,
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, dto.VerifyClientSignatureResponse{
		Valid: true,
	})
}
