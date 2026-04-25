package handler

import (
	"context"
	"errors"
	"net/http"
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

type DocumentHandler struct {
	uploadDocumentUseCase uploadDocumentUseCase
	sendDocumentUseCase   sendDocumentUseCase
}

func NewDocumentHandler(uploadDocumentUseCase uploadDocumentUseCase, sendDocumentUseCase sendDocumentUseCase) *DocumentHandler {
	return &DocumentHandler{
		uploadDocumentUseCase: uploadDocumentUseCase,
		sendDocumentUseCase:   sendDocumentUseCase,
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
		RecipientEmail: result.RecipientEmail,
		SendStatus:     result.SendStatus,
	}
	if result.SentAt != nil {
		response.SentAt = result.SentAt.Format(time.RFC3339Nano)
	}

	ctx.JSON(http.StatusOK, response)
}
