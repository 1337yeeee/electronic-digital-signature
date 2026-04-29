package usecase

import (
	"context"
	"fmt"
	"strings"

	"electronic-digital-signature/internal/domain/model"
)

type getDocumentDetailsRepository interface {
	FindByID(ctx context.Context, id string) (*model.Document, error)
}

type GetDocumentDetailsInput struct {
	DocumentID string
	UserID     string
}

type GetDocumentDetailsUseCase struct {
	repository getDocumentDetailsRepository
}

func NewGetDocumentDetailsUseCase(repository getDocumentDetailsRepository) *GetDocumentDetailsUseCase {
	return &GetDocumentDetailsUseCase{repository: repository}
}

func (uc *GetDocumentDetailsUseCase) Execute(ctx context.Context, input GetDocumentDetailsInput) (*model.Document, error) {
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
