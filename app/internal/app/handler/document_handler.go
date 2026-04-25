package handler

import (
	"context"
	"net/http"
	"time"

	"electronic-digital-signature/internal/app/dto"
	"electronic-digital-signature/internal/app/usecase"
	"electronic-digital-signature/internal/domain/model"

	"github.com/gin-gonic/gin"
)

type uploadDocumentUseCase interface {
	Execute(ctx context.Context, input usecase.UploadDocumentInput) (*model.Document, error)
}

type DocumentHandler struct {
	uploadDocumentUseCase uploadDocumentUseCase
}

func NewDocumentHandler(uploadDocumentUseCase uploadDocumentUseCase) *DocumentHandler {
	return &DocumentHandler{uploadDocumentUseCase: uploadDocumentUseCase}
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
