package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"electronic-digital-signature/internal/app/dto"
	"electronic-digital-signature/internal/app/usecase"
	"electronic-digital-signature/internal/domain/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type registerUserUseCase interface {
	Execute(ctx context.Context, input usecase.RegisterUserInput) (*model.User, error)
}

type getUserUseCase interface {
	Execute(ctx context.Context, id string) (*model.User, error)
}

type UserHandler struct {
	registerUserUseCase registerUserUseCase
	getUserUseCase      getUserUseCase
}

func NewUserHandler(registerUserUseCase registerUserUseCase, getUserUseCase getUserUseCase) *UserHandler {
	return &UserHandler{
		registerUserUseCase: registerUserUseCase,
		getUserUseCase:      getUserUseCase,
	}
}

func (h *UserHandler) Register(ctx *gin.Context) {
	if h.registerUserUseCase == nil {
		respondError(ctx, http.StatusInternalServerError, "internal_error", "User registration is not available right now.")
		return
	}

	var request dto.RegisterUserRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		respondError(ctx, http.StatusBadRequest, "invalid_request", "Request body is invalid.")
		return
	}

	user, err := h.registerUserUseCase.Execute(ctx.Request.Context(), usecase.RegisterUserInput{
		Email:        request.Email,
		Name:         request.Name,
		Password:     request.Password,
		PublicKeyPEM: request.PublicKeyPEM,
	})
	if err != nil {
		logRequestError(ctx, "register-user", err)
		switch {
		case errors.Is(err, usecase.ErrUserEmailAlreadyExists):
			respondError(ctx, http.StatusBadRequest, "email_already_exists", "User with this email already exists.")
		case errors.Is(err, usecase.ErrInvalidUserPublicKey):
			respondError(ctx, http.StatusBadRequest, "invalid_public_key", "Public key PEM is invalid.")
		case err.Error() == "email is required":
			respondError(ctx, http.StatusBadRequest, "email_required", "Email is required.")
		case err.Error() == "name is required":
			respondError(ctx, http.StatusBadRequest, "name_required", "Name is required.")
		case err.Error() == "password is required":
			respondError(ctx, http.StatusBadRequest, "password_required", "Password is required.")
		default:
			respondError(ctx, http.StatusBadRequest, "user_registration_failed", "User could not be registered.")
		}
		return
	}

	respondSuccess(ctx, http.StatusCreated, dto.UserResponse{
		ID:           user.ID,
		Email:        user.Email,
		Name:         user.Name,
		PublicKeyPEM: user.PublicKeyPEM,
		CreatedAt:    user.CreatedAt.Format(time.RFC3339Nano),
	})
}

func (h *UserHandler) GetByID(ctx *gin.Context) {
	if h.getUserUseCase == nil {
		respondError(ctx, http.StatusInternalServerError, "internal_error", "User lookup is not available right now.")
		return
	}

	user, err := h.getUserUseCase.Execute(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		logRequestError(ctx, "get-user", err)
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			respondError(ctx, http.StatusNotFound, "user_not_found", "User was not found.")
		case err.Error() == "user id is required":
			respondError(ctx, http.StatusBadRequest, "user_id_required", "User id is required.")
		default:
			respondError(ctx, http.StatusBadRequest, "user_lookup_failed", "User could not be loaded.")
		}
		return
	}

	respondSuccess(ctx, http.StatusOK, dto.UserResponse{
		ID:           user.ID,
		Email:        user.Email,
		Name:         user.Name,
		PublicKeyPEM: user.PublicKeyPEM,
		CreatedAt:    user.CreatedAt.Format(time.RFC3339Nano),
	})
}
