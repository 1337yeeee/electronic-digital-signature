package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"time"

	"electronic-digital-signature/internal/app/dto"
	"electronic-digital-signature/internal/app/usecase"
	"electronic-digital-signature/internal/domain/model"
	signaturecrypto "electronic-digital-signature/internal/infra/crypto"
	"electronic-digital-signature/internal/infra/keys"

	"github.com/gin-gonic/gin"
)

type signatureProvider interface {
	Verify(message []byte, signature []byte, publicKey []byte) error
	Hash(message []byte) []byte
	Sign(message []byte, privateKey []byte) ([]byte, error)
}

type SignatureHandler struct {
	serverKeys keys.ServerKeyPair
	provider   signatureProvider
}

func NewSignatureHandler(serverKeys keys.ServerKeyPair, provider signatureProvider) *SignatureHandler {
	return &SignatureHandler{
		serverKeys: serverKeys,
		provider:   provider,
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
		Algorithm:    signaturecrypto.ECDSASHA256Algorithm,
		PublicKeyPEM: string(h.serverKeys.PublicKey),
	})
}

func (h *SignatureHandler) VerifyClientSignature(ctx *gin.Context) {
	if h.provider == nil {
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
	if err := usecase.VerifyClientSignature(message, signature, []byte(request.PublicKey), h.provider); err != nil {
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

func (h *SignatureHandler) IssueServerMessage(ctx *gin.Context) {
	if h.provider == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "signature provider is not configured",
		})
		return
	}
	if len(h.serverKeys.PrivateKey) == 0 {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "server private key is not loaded",
		})
		return
	}

	var request dto.IssueServerMessageRequest
	if err := ctx.ShouldBindJSON(&request); err != nil && !errors.Is(err, io.EOF) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	messageText := request.Message
	if messageText == "" {
		messageText = randomServerMessage()
	}

	now := time.Now().UTC()
	message := model.Message{
		ID:        newRandomID(),
		Message:   messageText,
		CreatedAt: now,
	}

	signature, messageHash, err := usecase.IssueServerSignedMessage(&message, h.serverKeys.PrivateKey, h.provider)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, dto.IssueServerMessageResponse{
		ID:              message.ID,
		CreatedAt:       message.CreatedAt.Format(time.RFC3339Nano),
		Message:         message.Message,
		Algorithm:       signaturecrypto.ECDSASHA256Algorithm,
		HashBase64:      base64.StdEncoding.EncodeToString(messageHash),
		SignatureBase64: base64.StdEncoding.EncodeToString(signature),
	})
}

func randomServerMessage() string {
	return "server message " + newRandomID()
}

func newRandomID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}

	return hex.EncodeToString(bytes)
}
