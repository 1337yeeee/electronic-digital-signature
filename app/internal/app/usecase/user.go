package usecase

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"electronic-digital-signature/internal/domain/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserEmailAlreadyExists = errors.New("user email already exists")
	ErrInvalidUserPublicKey   = errors.New("invalid user public key")
)

type userRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	CreateKeyHistory(ctx context.Context, entry *model.UserKeyHistory) error
}

type userIDGenerator interface {
	Generate() (string, error)
}

type RegisterUserInput struct {
	Email        string
	Name         string
	Password     string
	PublicKeyPEM string
}

type RegisterUserUseCase struct {
	repository  userRepository
	idGenerator userIDGenerator
}

func NewRegisterUserUseCase(repository userRepository, idGenerator userIDGenerator) *RegisterUserUseCase {
	return &RegisterUserUseCase{
		repository:  repository,
		idGenerator: idGenerator,
	}
}

func (uc *RegisterUserUseCase) Execute(ctx context.Context, input RegisterUserInput) (*model.User, error) {
	if uc.repository == nil {
		return nil, fmt.Errorf("user repository is not configured")
	}
	if uc.idGenerator == nil {
		return nil, fmt.Errorf("user id generator is not configured")
	}

	email := strings.TrimSpace(strings.ToLower(input.Email))
	name := strings.TrimSpace(input.Name)
	password := strings.TrimSpace(input.Password)
	publicKeyPEM := strings.TrimSpace(input.PublicKeyPEM)

	if email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if password == "" {
		return nil, fmt.Errorf("password is required")
	}
	if publicKeyPEM != "" {
		if err := validateUserPublicKeyPEM(publicKeyPEM); err != nil {
			return nil, err
		}
	}

	if _, err := uc.repository.FindByEmail(ctx, email); err == nil {
		return nil, ErrUserEmailAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check existing user by email: %w", err)
	}

	id, err := uc.idGenerator.Generate()
	if err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	user := &model.User{
		ID:           id,
		Email:        email,
		Name:         name,
		PasswordHash: string(passwordHash),
		PublicKeyPEM: publicKeyPEM,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := uc.repository.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}
	if publicKeyPEM != "" {
		if err := uc.repository.CreateKeyHistory(ctx, &model.UserKeyHistory{
			UserID:       user.ID,
			PublicKeyPEM: publicKeyPEM,
			CreatedAt:    now,
		}); err != nil {
			return nil, fmt.Errorf("save user key history: %w", err)
		}
	}

	return user, nil
}

type GetUserUseCase struct {
	repository userRepository
}

func NewGetUserUseCase(repository userRepository) *GetUserUseCase {
	return &GetUserUseCase{repository: repository}
}

func (uc *GetUserUseCase) Execute(ctx context.Context, id string) (*model.User, error) {
	if uc.repository == nil {
		return nil, fmt.Errorf("user repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("user id is required")
	}

	return uc.repository.FindByID(ctx, id)
}

func validateUserPublicKeyPEM(publicKeyPEM string) error {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return ErrInvalidUserPublicKey
	}

	parsedKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return ErrInvalidUserPublicKey
	}
	if _, ok := parsedKey.(*ecdsa.PublicKey); !ok {
		return ErrInvalidUserPublicKey
	}

	return nil
}

type UpdateCurrentUserPublicKeyInput struct {
	UserID       string
	PublicKeyPEM string
}

type UpdateCurrentUserPublicKeyUseCase struct {
	repository userRepository
}

func NewUpdateCurrentUserPublicKeyUseCase(repository userRepository) *UpdateCurrentUserPublicKeyUseCase {
	return &UpdateCurrentUserPublicKeyUseCase{repository: repository}
}

func (uc *UpdateCurrentUserPublicKeyUseCase) Execute(ctx context.Context, input UpdateCurrentUserPublicKeyInput) (*model.User, error) {
	if uc.repository == nil {
		return nil, fmt.Errorf("user repository is not configured")
	}

	userID := strings.TrimSpace(input.UserID)
	publicKeyPEM := strings.TrimSpace(input.PublicKeyPEM)
	if userID == "" {
		return nil, fmt.Errorf("user id is required")
	}
	if publicKeyPEM == "" {
		return nil, fmt.Errorf("public key pem is required")
	}
	if err := validateUserPublicKeyPEM(publicKeyPEM); err != nil {
		return nil, err
	}

	user, err := uc.repository.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.PublicKeyPEM == publicKeyPEM {
		return user, nil
	}

	now := time.Now().UTC()
	user.PublicKeyPEM = publicKeyPEM
	user.UpdatedAt = now
	if err := uc.repository.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user public key: %w", err)
	}
	if err := uc.repository.CreateKeyHistory(ctx, &model.UserKeyHistory{
		UserID:       user.ID,
		PublicKeyPEM: publicKeyPEM,
		CreatedAt:    now,
	}); err != nil {
		return nil, fmt.Errorf("save user key history: %w", err)
	}

	return user, nil
}
