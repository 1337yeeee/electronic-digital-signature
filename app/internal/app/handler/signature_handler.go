package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"electronic-digital-signature/internal/app/dto"
	"electronic-digital-signature/internal/domain/model"
	signaturecrypto "electronic-digital-signature/internal/infra/crypto"
	"electronic-digital-signature/internal/infra/keys"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type verifyClientSignatureUseCase interface {
	Execute(message model.Message, signature []byte, publicKey []byte) error
}

type issueServerSignedMessageUseCase interface {
	Execute(ctx context.Context, message *model.Message) (signature []byte, messageHash []byte, err error)
}

type getServerSignedMessageUseCase interface {
	Execute(ctx context.Context, id string) (*model.Message, error)
}

type SignatureHandler struct {
	serverKeys                      keys.ServerKeyPair
	verifyClientSignatureUseCase    verifyClientSignatureUseCase
	issueServerSignedMessageUseCase issueServerSignedMessageUseCase
	getServerSignedMessageUseCase   getServerSignedMessageUseCase
}

func NewSignatureHandler(
	serverKeys keys.ServerKeyPair,
	verifyClientSignatureUseCase verifyClientSignatureUseCase,
	issueServerSignedMessageUseCase issueServerSignedMessageUseCase,
	getServerSignedMessageUseCase getServerSignedMessageUseCase,
) *SignatureHandler {
	return &SignatureHandler{
		serverKeys:                      serverKeys,
		verifyClientSignatureUseCase:    verifyClientSignatureUseCase,
		issueServerSignedMessageUseCase: issueServerSignedMessageUseCase,
		getServerSignedMessageUseCase:   getServerSignedMessageUseCase,
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
	if h.verifyClientSignatureUseCase == nil {
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
	if strings.TrimSpace(request.Message) == "" {
		ctx.JSON(http.StatusBadRequest, dto.VerifyClientSignatureResponse{
			Valid: false,
			Error: "message is required",
		})
		return
	}
	if strings.TrimSpace(request.PublicKey) == "" {
		ctx.JSON(http.StatusBadRequest, dto.VerifyClientSignatureResponse{
			Valid: false,
			Error: "public_key is required",
		})
		return
	}
	if strings.TrimSpace(request.SignatureBase64) == "" {
		ctx.JSON(http.StatusBadRequest, dto.VerifyClientSignatureResponse{
			Valid: false,
			Error: "signature_base64 is required",
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
	if err := h.verifyClientSignatureUseCase.Execute(message, signature, []byte(request.PublicKey)); err != nil {
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
	if h.issueServerSignedMessageUseCase == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "issue server signed message usecase is not configured",
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
	currentUser, ok := currentUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "authentication is required"})
		return
	}

	message := model.Message{
		Message:         messageText,
		CreatedByUserID: currentUser.ID,
	}

	signature, messageHash, err := h.issueServerSignedMessageUseCase.Execute(ctx.Request.Context(), &message)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, dto.IssueServerMessageResponse{
		MessageID:       message.ID,
		CreatedByUserID: message.CreatedByUserID,
		CreatedAt:       message.CreatedAt.Format(time.RFC3339Nano),
		Message:         message.Message,
		Algorithm:       signaturecrypto.ECDSASHA256Algorithm,
		HashBase64:      base64.StdEncoding.EncodeToString(messageHash),
		SignatureBase64: base64.StdEncoding.EncodeToString(signature),
	})
}

func (h *SignatureHandler) GetServerMessage(ctx *gin.Context) {
	if h.getServerSignedMessageUseCase == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "get server signed message usecase is not configured",
		})
		return
	}

	messageID := ctx.Param("id")
	if messageID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "message id is required"})
		return
	}

	message, err := h.getServerSignedMessageUseCase.Execute(ctx.Request.Context(), messageID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, dto.IssueServerMessageResponse{
		MessageID:       message.ID,
		CreatedByUserID: message.CreatedByUserID,
		CreatedAt:       message.CreatedAt.Format(time.RFC3339Nano),
		Message:         message.Message,
		Algorithm:       signaturecrypto.ECDSASHA256Algorithm,
		HashBase64:      base64.StdEncoding.EncodeToString(message.Hash),
		SignatureBase64: base64.StdEncoding.EncodeToString(message.Signature),
	})
}

func randomServerMessage() string {
	return "server message " + time.Now().UTC().Format(time.RFC3339Nano)
}
