package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"electronic-digital-signature/internal/domain/model"
	"electronic-digital-signature/internal/infra/auth"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type loginUserRepository interface {
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id string) (*model.User, error)
}

type tokenManager interface {
	Generate(subject, email string) (string, time.Time, error)
	Verify(token string) (auth.Claims, error)
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginResult struct {
	AccessToken string
	ExpiresAt   time.Time
	User        *model.User
}

type LoginUseCase struct {
	repository   loginUserRepository
	tokenManager tokenManager
}

func NewLoginUseCase(repository loginUserRepository, tokenManager tokenManager) *LoginUseCase {
	return &LoginUseCase{
		repository:   repository,
		tokenManager: tokenManager,
	}
}

func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginResult, error) {
	if uc.repository == nil {
		return nil, fmt.Errorf("user repository is not configured")
	}
	if uc.tokenManager == nil {
		return nil, fmt.Errorf("token manager is not configured")
	}

	email := strings.TrimSpace(strings.ToLower(input.Email))
	password := strings.TrimSpace(input.Password)
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if password == "" {
		return nil, fmt.Errorf("password is required")
	}

	user, err := uc.repository.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, expiresAt, err := uc.tokenManager.Generate(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	return &LoginResult{
		AccessToken: accessToken,
		ExpiresAt:   expiresAt,
		User:        user,
	}, nil
}

type CurrentUserUseCase struct {
	repository loginUserRepository
}

func NewCurrentUserUseCase(repository loginUserRepository) *CurrentUserUseCase {
	return &CurrentUserUseCase{repository: repository}
}

func (uc *CurrentUserUseCase) Execute(ctx context.Context, userID string) (*model.User, error) {
	if uc.repository == nil {
		return nil, fmt.Errorf("user repository is not configured")
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("user id is required")
	}

	return uc.repository.FindByID(ctx, userID)
}
