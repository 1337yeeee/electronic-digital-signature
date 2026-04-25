package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"electronic-digital-signature/internal/app/dto"
	"electronic-digital-signature/internal/app/usecase"
	"electronic-digital-signature/internal/domain/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type uploadDocumentUseCase interface {
	Execute(ctx context.Context, input usecase.UploadDocumentInput) (*model.Document, error)
}

type sendDocumentUseCase interface {
	Execute(ctx context.Context, input usecase.SendDocumentInput) (*usecase.SendDocumentResult, error)
}

type verifyDecryptPackageUseCase interface {
	Execute(ctx context.Context, input usecase.VerifyDecryptPackageInput) (*usecase.VerifyDecryptPackageResult, error)
}

type DocumentHandler struct {
	uploadDocumentUseCase       uploadDocumentUseCase
	sendDocumentUseCase         sendDocumentUseCase
	verifyDecryptPackageUseCase verifyDecryptPackageUseCase
}

func NewDocumentHandler(
	uploadDocumentUseCase uploadDocumentUseCase,
	sendDocumentUseCase sendDocumentUseCase,
	verifyDecryptPackageUseCase verifyDecryptPackageUseCase,
) *DocumentHandler {
	return &DocumentHandler{
		uploadDocumentUseCase:       uploadDocumentUseCase,
		sendDocumentUseCase:         sendDocumentUseCase,
		verifyDecryptPackageUseCase: verifyDecryptPackageUseCase,
	}
}

func (h *DocumentHandler) UploadDocument(ctx *gin.Context) {
	if h.uploadDocumentUseCase == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "upload document usecase is not configured"})
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "document file is required"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "open uploaded document file"})
		return
	}
	defer file.Close()

	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}

	document, err := h.uploadDocumentUseCase.Execute(ctx.Request.Context(), usecase.UploadDocumentInput{
		OwnerEmail:       ctx.PostForm("owner_email"),
		RecipientEmail:   ctx.PostForm("recipient_email"),
		OriginalFileName: fileHeader.Filename,
		MimeType:         mimeType,
		Content:          file,
	})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, dto.UploadDocumentResponse{
		DocumentID:       document.ID,
		OwnerEmail:       document.OwnerEmail,
		RecipientEmail:   document.RecipientEmail,
		OriginalFileName: document.OriginalFileName,
		StoredPath:       document.StoredPath,
		MimeType:         document.MimeType,
		CreatedAt:        document.CreatedAt.Format(time.RFC3339Nano),
	})
}

func (h *DocumentHandler) SendDocument(ctx *gin.Context) {
	if h.sendDocumentUseCase == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "send document usecase is not configured"})
		return
	}

	var request dto.SendDocumentRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	result, err := h.sendDocumentUseCase.Execute(ctx.Request.Context(), usecase.SendDocumentInput{
		DocumentID:     ctx.Param("id"),
		RecipientEmail: request.Email,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := dto.SendDocumentResponse{
		DocumentID:     result.DocumentID,
		PackageID:      result.PackageID,
		RecipientEmail: result.RecipientEmail,
		SendStatus:     result.SendStatus,
	}
	if result.SentAt != nil {
		response.SentAt = result.SentAt.Format(time.RFC3339Nano)
	}

	ctx.JSON(http.StatusOK, response)
}

func (h *DocumentHandler) VerifyDecryptPackage(ctx *gin.Context) {
	if h.verifyDecryptPackageUseCase == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "verify decrypt package usecase is not configured"})
		return
	}

	packageContent, err := readPackageContent(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.verifyDecryptPackageUseCase.Execute(ctx.Request.Context(), usecase.VerifyDecryptPackageInput{
		PackageContent: packageContent,
	})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := dto.VerifyDecryptPackageResponse{
		Valid: true,
		Metadata: dto.VerifyDecryptPackageMetadata{
			DocumentID:          result.Metadata.DocumentID,
			Version:             result.Metadata.Version,
			EncryptionAlgorithm: result.Metadata.EncryptionAlgorithm,
			KeyTransport:        result.Metadata.KeyTransport,
			SignatureAlgorithm:  result.Metadata.SignatureAlgorithm,
			OriginalFileName:    result.Metadata.OriginalFileName,
			MimeType:            result.Metadata.MimeType,
			HashBase64:          result.Metadata.HashBase64,
		},
		DecryptedDocumentBase64: base64.StdEncoding.EncodeToString(result.DecryptedDocument),
	}

	ctx.JSON(http.StatusOK, response)
}

func readPackageContent(ctx *gin.Context) ([]byte, error) {
	contentType := ctx.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		fileHeader, err := ctx.FormFile("package")
		if err != nil {
			fileHeader, err = ctx.FormFile("file")
			if err != nil {
				return nil, fmt.Errorf("package file is required")
			}
		}

		file, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("open package file")
		}
		defer file.Close()

		return io.ReadAll(file)
	}

	content, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("read package body: %w", err)
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("package json is required")
	}

	return content, nil
}
