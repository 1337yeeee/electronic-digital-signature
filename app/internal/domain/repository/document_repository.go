package repository

import (
	"context"
	"errors"

	"electronic-digital-signature/internal/domain/model"

	"gorm.io/gorm"
)

type DocumentRepository struct {
	db *gorm.DB
}

func NewDocumentRepository(db *gorm.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

func (r *DocumentRepository) Create(ctx context.Context, document *model.Document) error {
	if r.db == nil {
		return errors.New("document repository db is not configured")
	}

	return r.db.WithContext(ctx).Create(document).Error
}

func (r *DocumentRepository) FindByID(ctx context.Context, id string) (*model.Document, error) {
	if r.db == nil {
		return nil, errors.New("document repository db is not configured")
	}

	var document model.Document
	if err := r.db.WithContext(ctx).First(&document, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &document, nil
}
