package usecase

import (
	"context"
	"fmt"
	"strings"

	"electronic-digital-signature/internal/domain/model"
)

type listUserDocumentsRepository interface {
	ListByOwnerUserID(ctx context.Context, ownerUserID string) ([]model.Document, error)
}

type ListUserDocumentsInput struct {
	UserID string
}

type ListUserDocumentsUseCase struct {
	repository listUserDocumentsRepository
}

func NewListUserDocumentsUseCase(repository listUserDocumentsRepository) *ListUserDocumentsUseCase {
	return &ListUserDocumentsUseCase{repository: repository}
}

func (uc *ListUserDocumentsUseCase) Execute(ctx context.Context, input ListUserDocumentsInput) ([]model.Document, error) {
	if uc.repository == nil {
		return nil, fmt.Errorf("document repository is not configured")
	}
	if strings.TrimSpace(input.UserID) == "" {
		return nil, fmt.Errorf("user id is required")
	}

	return uc.repository.ListByOwnerUserID(ctx, input.UserID)
}
