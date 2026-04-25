package usecase

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"electronic-digital-signature/internal/domain/model"
)

type documentRepository interface {
	Create(ctx context.Context, document *model.Document) error
}

type documentStorage interface {
	Save(ctx context.Context, id, originalFileName string, content io.Reader) (string, error)
}

type documentIDGenerator interface {
	Generate() (string, error)
}

type UploadDocumentInput struct {
	OwnerEmail       string
	RecipientEmail   string
	OriginalFileName string
	MimeType         string
	Content          io.Reader
}

type UploadDocumentUseCase struct {
	repository  documentRepository
	storage     documentStorage
	idGenerator documentIDGenerator
}

func NewUploadDocumentUseCase(repository documentRepository, storage documentStorage, idGenerator documentIDGenerator) *UploadDocumentUseCase {
	return &UploadDocumentUseCase{
		repository:  repository,
		storage:     storage,
		idGenerator: idGenerator,
	}
}

func (uc *UploadDocumentUseCase) Execute(ctx context.Context, input UploadDocumentInput) (*model.Document, error) {
	if uc.repository == nil {
		return nil, fmt.Errorf("document repository is not configured")
	}
	if uc.storage == nil {
		return nil, fmt.Errorf("document storage is not configured")
	}
	if uc.idGenerator == nil {
		return nil, fmt.Errorf("document id generator is not configured")
	}
	if input.Content == nil {
		return nil, fmt.Errorf("document file is required")
	}
	if !strings.EqualFold(filepath.Ext(input.OriginalFileName), ".docx") {
		return nil, fmt.Errorf("document file must have .docx extension")
	}

	id, err := uc.idGenerator.Generate()
	if err != nil {
		return nil, err
	}

	storedPath, err := uc.storage.Save(ctx, id, input.OriginalFileName, input.Content)
	if err != nil {
		return nil, err
	}

	document := &model.Document{
		ID:               id,
		OwnerEmail:       input.OwnerEmail,
		RecipientEmail:   input.RecipientEmail,
		OriginalFileName: input.OriginalFileName,
		StoredPath:       storedPath,
		MimeType:         input.MimeType,
		CreatedAt:        time.Now().UTC(),
	}

	if err := uc.repository.Create(ctx, document); err != nil {
		return nil, fmt.Errorf("save uploaded document: %w", err)
	}

	return document, nil
}
