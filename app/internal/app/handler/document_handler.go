package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
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

type getDocumentAuditUseCase interface {
	Execute(ctx context.Context, input usecase.GetDocumentAuditInput) (*model.Document, error)
}

type listUserDocumentsUseCase interface {
	Execute(ctx context.Context, input usecase.ListUserDocumentsInput) ([]model.Document, error)
}

type verifyDecryptPackageUseCase interface {
	Execute(ctx context.Context, input usecase.VerifyDecryptPackageInput) (*usecase.VerifyDecryptPackageResult, error)
}

type DocumentHandler struct {
	uploadDocumentUseCase       uploadDocumentUseCase
	sendDocumentUseCase         sendDocumentUseCase
	getDocumentAuditUseCase     getDocumentAuditUseCase
	listUserDocumentsUseCase    listUserDocumentsUseCase
	verifyDecryptPackageUseCase verifyDecryptPackageUseCase
}

func NewDocumentHandler(
	uploadDocumentUseCase uploadDocumentUseCase,
	sendDocumentUseCase sendDocumentUseCase,
	getDocumentAuditUseCase getDocumentAuditUseCase,
	listUserDocumentsUseCase listUserDocumentsUseCase,
	verifyDecryptPackageUseCase verifyDecryptPackageUseCase,
) *DocumentHandler {
	return &DocumentHandler{
		uploadDocumentUseCase:       uploadDocumentUseCase,
		sendDocumentUseCase:         sendDocumentUseCase,
		getDocumentAuditUseCase:     getDocumentAuditUseCase,
		listUserDocumentsUseCase:    listUserDocumentsUseCase,
		verifyDecryptPackageUseCase: verifyDecryptPackageUseCase,
	}
}

func (h *DocumentHandler) UploadDocument(ctx *gin.Context) {
	if h.uploadDocumentUseCase == nil {
		respondError(ctx, http.StatusInternalServerError, "internal_error", "Document upload is not available right now.")
		return
	}
	currentUser, ok := currentUserFromContext(ctx)
	if !ok {
		respondError(ctx, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		respondError(ctx, http.StatusBadRequest, "document_file_required", "Document file is required.")
		return
	}
	if fileHeader.Size <= 0 {
		respondError(ctx, http.StatusBadRequest, "document_file_required", "Document file is required.")
		return
	}
	if fileHeader.Size > usecase.MaxUploadDocumentSizeBytes {
		respondError(ctx, http.StatusBadRequest, "document_too_large", "Document file exceeds the maximum allowed size.")
		return
	}
	if !strings.EqualFold(fileExtension(fileHeader.Filename), ".docx") {
		respondError(ctx, http.StatusBadRequest, "invalid_document_type", "Document file must have .docx extension.")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		logRequestError(ctx, "upload-document-open-file", err)
		respondError(ctx, http.StatusBadRequest, "document_file_unreadable", "Uploaded document file could not be opened.")
		return
	}
	defer file.Close()

	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}
	if !isAllowedUploadDocumentMIMEType(mimeType) {
		respondError(ctx, http.StatusBadRequest, "invalid_document_type", "Document MIME type is not supported.")
		return
	}

	document, err := h.uploadDocumentUseCase.Execute(ctx.Request.Context(), usecase.UploadDocumentInput{
		OwnerUserID:      currentUser.ID,
		SignedByUserID:   currentUser.ID,
		OwnerEmail:       currentUser.Email,
		RecipientEmail:   ctx.PostForm("recipient_email"),
		OriginalFileName: fileHeader.Filename,
		MimeType:         mimeType,
		Content:          file,
	})
	if err != nil {
		logRequestError(ctx, "upload-document", err)
		if strings.Contains(err.Error(), ".docx extension") {
			respondError(ctx, http.StatusBadRequest, "invalid_document_type", "Document file must have .docx extension.")
			return
		}
		if strings.Contains(err.Error(), "document file is required") {
			respondError(ctx, http.StatusBadRequest, "document_file_required", "Document file is required.")
			return
		}
		respondError(ctx, http.StatusBadRequest, "document_upload_failed", "Document could not be uploaded.")
		return
	}

	respondSuccess(ctx, http.StatusCreated, dto.UploadDocumentResponse{
		DocumentID:       document.ID,
		OwnerUserID:      document.OwnerUserID,
		SignedByUserID:   document.SignedByUserID,
		OwnerEmail:       document.OwnerEmail,
		RecipientEmail:   document.RecipientEmail,
		OriginalFileName: document.OriginalFileName,
		StoredPath:       document.StoredPath,
		MimeType:         document.MimeType,
		CreatedAt:        document.CreatedAt.Format(time.RFC3339Nano),
	})
	logRequestInfo(ctx, "upload-document", fmt.Sprintf("document_id=%s owner_user_id=%s signed_by_user_id=%s", document.ID, document.OwnerUserID, document.SignedByUserID))
}

func (h *DocumentHandler) SendDocument(ctx *gin.Context) {
	if h.sendDocumentUseCase == nil {
		respondError(ctx, http.StatusInternalServerError, "internal_error", "Document sending is not available right now.")
		return
	}
	currentUser, ok := currentUserFromContext(ctx)
	if !ok {
		respondError(ctx, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	var request dto.SendDocumentRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		respondError(ctx, http.StatusBadRequest, "invalid_request", "Request body is invalid.")
		return
	}

	result, err := h.sendDocumentUseCase.Execute(ctx.Request.Context(), usecase.SendDocumentInput{
		DocumentID:     ctx.Param("id"),
		RecipientEmail: request.Email,
		SentByUserID:   currentUser.ID,
	})
	if err != nil {
		logRequestError(ctx, "send-document", err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(ctx, http.StatusNotFound, "document_not_found", "Document was not found.")
			return
		}
		if errors.Is(err, usecase.ErrDocumentAccessDenied) {
			respondError(ctx, http.StatusForbidden, "forbidden", "You do not have access to this document.")
			return
		}
		if strings.Contains(err.Error(), "recipient email is required") {
			respondError(ctx, http.StatusBadRequest, "recipient_email_required", "Recipient email is required.")
			return
		}
		respondError(ctx, http.StatusBadRequest, "document_send_failed", "Document package could not be sent.")
		return
	}

	response := dto.SendDocumentResponse{
		DocumentID:     result.DocumentID,
		OwnerUserID:    result.OwnerUserID,
		SignedByUserID: result.SignedByUserID,
		PackageID:      result.PackageID,
		RecipientEmail: result.RecipientEmail,
		SendStatus:     result.SendStatus,
		SentByUserID:   result.SentByUserID,
	}
	if result.SentAt != nil {
		response.SentAt = result.SentAt.Format(time.RFC3339Nano)
	}

	respondSuccess(ctx, http.StatusOK, response)
	logRequestInfo(ctx, "send-document", fmt.Sprintf("document_id=%s owner_user_id=%s signed_by_user_id=%s sent_by_user_id=%s", result.DocumentID, result.OwnerUserID, result.SignedByUserID, result.SentByUserID))
}

func (h *DocumentHandler) GetAudit(ctx *gin.Context) {
	if h.getDocumentAuditUseCase == nil {
		respondError(ctx, http.StatusInternalServerError, "internal_error", "Document audit is not available right now.")
		return
	}

	currentUser, ok := currentUserFromContext(ctx)
	if !ok {
		respondError(ctx, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	document, err := h.getDocumentAuditUseCase.Execute(ctx.Request.Context(), usecase.GetDocumentAuditInput{
		DocumentID: ctx.Param("id"),
		UserID:     currentUser.ID,
	})
	if err != nil {
		logRequestError(ctx, "get-document-audit", err)
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			respondError(ctx, http.StatusNotFound, "document_not_found", "Document was not found.")
		case errors.Is(err, usecase.ErrDocumentAccessDenied):
			respondError(ctx, http.StatusForbidden, "forbidden", "You do not have access to this document.")
		case err.Error() == "document_id is required":
			respondError(ctx, http.StatusBadRequest, "document_id_required", "Document id is required.")
		default:
			respondError(ctx, http.StatusBadRequest, "document_audit_failed", "Document audit could not be loaded.")
		}
		return
	}

	response := dto.DocumentAuditResponse{
		DocumentID:       document.ID,
		OwnerUserID:      document.OwnerUserID,
		SignedByUserID:   document.SignedByUserID,
		SentByUserID:     document.LastSentByUserID,
		OwnerEmail:       document.OwnerEmail,
		RecipientEmail:   document.RecipientEmail,
		OriginalFileName: document.OriginalFileName,
		MimeType:         document.MimeType,
		SendStatus:       document.SendStatus,
		CreatedAt:        document.CreatedAt.Format(time.RFC3339Nano),
		SignedAt:         document.SignedAt.Format(time.RFC3339Nano),
	}
	if document.SentAt != nil {
		response.SentAt = document.SentAt.Format(time.RFC3339Nano)
	}

	respondSuccess(ctx, http.StatusOK, response)
	logRequestInfo(ctx, "get-document-audit", fmt.Sprintf("document_id=%s owner_user_id=%s signed_by_user_id=%s sent_by_user_id=%s", document.ID, document.OwnerUserID, document.SignedByUserID, document.LastSentByUserID))
}

func (h *DocumentHandler) ListMyDocuments(ctx *gin.Context) {
	if h.listUserDocumentsUseCase == nil {
		respondError(ctx, http.StatusInternalServerError, "internal_error", "Document list is not available right now.")
		return
	}

	currentUser, ok := currentUserFromContext(ctx)
	if !ok {
		respondError(ctx, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	documents, err := h.listUserDocumentsUseCase.Execute(ctx.Request.Context(), usecase.ListUserDocumentsInput{
		UserID: currentUser.ID,
	})
	if err != nil {
		logRequestError(ctx, "list-my-documents", err)
		respondError(ctx, http.StatusBadRequest, "document_list_failed", "Document list could not be loaded.")
		return
	}

	response := make([]dto.UserDocumentListItemResponse, 0, len(documents))
	for _, document := range documents {
		response = append(response, dto.UserDocumentListItemResponse{
			DocumentID:       document.ID,
			OriginalFileName: document.OriginalFileName,
			RecipientEmail:   document.RecipientEmail,
			SendStatus:       document.SendStatus,
			CreatedAt:        document.CreatedAt.Format(time.RFC3339Nano),
		})
	}

	respondSuccess(ctx, http.StatusOK, response)
	logRequestInfo(ctx, "list-my-documents", fmt.Sprintf("user_id=%s documents=%d", currentUser.ID, len(response)))
}

func (h *DocumentHandler) VerifyDecryptPackage(ctx *gin.Context) {
	if h.verifyDecryptPackageUseCase == nil {
		respondError(ctx, http.StatusInternalServerError, "internal_error", "Package verification is not available right now.")
		return
	}

	packageContent, err := readPackageContent(ctx)
	if err != nil {
		respondError(ctx, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.verifyDecryptPackageUseCase.Execute(ctx.Request.Context(), usecase.VerifyDecryptPackageInput{
		PackageContent: packageContent,
	})
	if err != nil {
		logRequestError(ctx, "verify-decrypt-package", err)
		switch {
		case errors.Is(err, usecase.ErrInvalidSignature):
			respondError(ctx, http.StatusBadRequest, "invalid_signature", "Package signature is invalid.")
		case errors.Is(err, usecase.ErrInvalidEncryptedPackage):
			respondError(ctx, http.StatusBadRequest, "invalid_package", "Encrypted package is invalid.")
		default:
			respondError(ctx, http.StatusBadRequest, "verify_decrypt_failed", "Package could not be verified.")
		}
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

	respondSuccess(ctx, http.StatusOK, response)
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

func isAllowedUploadDocumentMIMEType(mimeType string) bool {
	_, ok := usecase.AllowedUploadDocumentMIMETypes[strings.ToLower(strings.TrimSpace(mimeType))]
	return ok
}

func fileExtension(fileName string) string {
	return strings.ToLower(strings.TrimSpace(filepath.Ext(fileName)))
}
