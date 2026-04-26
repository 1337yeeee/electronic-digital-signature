package usecase

import (
	"context"
	"fmt"
	"strings"

	"electronic-digital-signature/internal/domain/model"
)

type getDocumentAuditRepository interface {
	FindByID(ctx context.Context, id string) (*model.Document, error)
}

type GetDocumentAuditInput struct {
	DocumentID string
	UserID     string
}

type GetDocumentAuditUseCase struct {
	repository getDocumentAuditRepository
}

func NewGetDocumentAuditUseCase(repository getDocumentAuditRepository) *GetDocumentAuditUseCase {
	return &GetDocumentAuditUseCase{repository: repository}
}

func (uc *GetDocumentAuditUseCase) Execute(ctx context.Context, input GetDocumentAuditInput) (*model.Document, error) {
	if uc.repository == nil {
		return nil, fmt.Errorf("document repository is not configured")
	}
	if strings.TrimSpace(input.DocumentID) == "" {
		return nil, fmt.Errorf("document_id is required")
	}
	if strings.TrimSpace(input.UserID) == "" {
		return nil, fmt.Errorf("user id is required")
	}

	document, err := uc.repository.FindByID(ctx, input.DocumentID)
	if err != nil {
		return nil, err
	}
	if document.OwnerUserID != input.UserID {
		return nil, ErrDocumentAccessDenied
	}

	return document, nil
}
