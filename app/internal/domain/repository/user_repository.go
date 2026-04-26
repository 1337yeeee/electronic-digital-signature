package repository

import (
	"context"
	"errors"

	"electronic-digital-signature/internal/domain/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	if r.db == nil {
		return errors.New("user repository db is not configured")
	}

	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	if r.db == nil {
		return nil, errors.New("user repository db is not configured")
	}

	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	if r.db == nil {
		return nil, errors.New("user repository db is not configured")
	}

	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	if r.db == nil {
		return errors.New("user repository db is not configured")
	}

	return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepository) CreateKeyHistory(ctx context.Context, entry *model.UserKeyHistory) error {
	if r.db == nil {
		return errors.New("user repository db is not configured")
	}

	return r.db.WithContext(ctx).Create(entry).Error
}
