package repository

import (
	"context"
	"errors"

	"electronic-digital-signature/internal/domain/model"

	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(ctx context.Context, message *model.Message) error {
	if r.db == nil {
		return errors.New("message repository db is not configured")
	}

	return r.db.WithContext(ctx).Create(message).Error
}

func (r *MessageRepository) FindByID(ctx context.Context, id string) (*model.Message, error) {
	if r.db == nil {
		return nil, errors.New("message repository db is not configured")
	}

	var message model.Message
	if err := r.db.WithContext(ctx).First(&message, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &message, nil
}

func (r *MessageRepository) List(ctx context.Context, limit, offset int) ([]model.Message, error) {
	if r.db == nil {
		return nil, errors.New("message repository db is not configured")
	}

	var messages []model.Message
	query := r.db.WithContext(ctx).Order("created_at desc")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&messages).Error; err != nil {
		return nil, err
	}

	return messages, nil
}
