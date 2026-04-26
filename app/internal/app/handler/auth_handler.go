package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"electronic-digital-signature/internal/app/dto"
	"electronic-digital-signature/internal/app/usecase"
	"electronic-digital-signature/internal/domain/model"
	"electronic-digital-signature/internal/infra/auth"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const currentUserContextKey = "current_user"

type loginUseCase interface {
	Execute(ctx context.Context, input usecase.LoginInput) (*usecase.LoginResult, error)
}

type currentUserUseCase interface {
	Execute(ctx context.Context, userID string) (*model.User, error)
}

type authTokenManager interface {
	Verify(token string) (auth.Claims, error)
}

type AuthHandler struct {
	loginUseCase       loginUseCase
	currentUserUseCase currentUserUseCase
}

func NewAuthHandler(loginUseCase loginUseCase, currentUserUseCase currentUserUseCase) *AuthHandler {
	return &AuthHandler{
		loginUseCase:       loginUseCase,
		currentUserUseCase: currentUserUseCase,
	}
}

func (h *AuthHandler) Login(ctx *gin.Context) {
	if h.loginUseCase == nil {
		respondError(ctx, http.StatusInternalServerError, "internal_error", "Authentication is not available right now.")
		return
	}

	var request dto.LoginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		respondError(ctx, http.StatusBadRequest, "invalid_request", "Request body is invalid.")
		return
	}

	result, err := h.loginUseCase.Execute(ctx.Request.Context(), usecase.LoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		logRequestError(ctx, "auth-login", err)
		switch {
		case errors.Is(err, usecase.ErrInvalidCredentials):
			respondError(ctx, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect.")
		case err.Error() == "email is required":
			respondError(ctx, http.StatusBadRequest, "email_required", "Email is required.")
		case err.Error() == "password is required":
			respondError(ctx, http.StatusBadRequest, "password_required", "Password is required.")
		default:
			respondError(ctx, http.StatusBadRequest, "login_failed", "Login could not be completed.")
		}
		return
	}

	respondSuccess(ctx, http.StatusOK, dto.LoginResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresAt:   result.ExpiresAt.Format(time.RFC3339Nano),
		User:        userToDTO(result.User),
	})
}

func (h *AuthHandler) Me(ctx *gin.Context) {
	currentUser, ok := currentUserFromContext(ctx)
	if !ok {
		respondError(ctx, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}

	respondSuccess(ctx, http.StatusOK, userToDTO(currentUser))
}

type AuthMiddleware struct {
	tokenManager       authTokenManager
	currentUserUseCase currentUserUseCase
}

func NewAuthMiddleware(tokenManager authTokenManager, currentUserUseCase currentUserUseCase) *AuthMiddleware {
	return &AuthMiddleware{
		tokenManager:       tokenManager,
		currentUserUseCase: currentUserUseCase,
	}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if m.tokenManager == nil || m.currentUserUseCase == nil {
			respondError(ctx, http.StatusInternalServerError, "internal_error", "Authentication is not available right now.")
			ctx.Abort()
			return
		}

		authHeader := strings.TrimSpace(ctx.GetHeader("Authorization"))
		if !strings.HasPrefix(authHeader, "Bearer ") {
			respondError(ctx, http.StatusUnauthorized, "unauthorized", "Bearer token is required.")
			ctx.Abort()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		claims, err := m.tokenManager.Verify(token)
		if err != nil {
			logRequestError(ctx, "auth-middleware-verify-token", err)
			switch {
			case errors.Is(err, auth.ErrExpiredToken):
				respondError(ctx, http.StatusUnauthorized, "token_expired", "Access token has expired.")
			default:
				respondError(ctx, http.StatusUnauthorized, "invalid_token", "Access token is invalid.")
			}
			ctx.Abort()
			return
		}

		user, err := m.currentUserUseCase.Execute(ctx.Request.Context(), claims.Subject)
		if err != nil {
			logRequestError(ctx, "auth-middleware-load-user", err)
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				respondError(ctx, http.StatusUnauthorized, "invalid_token", "Access token is invalid.")
			default:
				respondError(ctx, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
			}
			ctx.Abort()
			return
		}

		ctx.Set(currentUserContextKey, user)
		ctx.Next()
	}
}

func currentUserFromContext(ctx *gin.Context) (*model.User, bool) {
	value, ok := ctx.Get(currentUserContextKey)
	if !ok {
		return nil, false
	}

	user, ok := value.(*model.User)
	return user, ok
}

func userToDTO(user *model.User) dto.UserResponse {
	return dto.UserResponse{
		ID:           user.ID,
		Email:        user.Email,
		Name:         user.Name,
		PublicKeyPEM: user.PublicKeyPEM,
		CreatedAt:    user.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:    user.UpdatedAt.Format(time.RFC3339Nano),
	}
}
