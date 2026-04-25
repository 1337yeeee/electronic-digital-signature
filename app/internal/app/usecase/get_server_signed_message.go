package usecase

import (
	"context"

	"electronic-digital-signature/internal/domain/model"
)

type serverMessageFinder interface {
	FindByID(ctx context.Context, id string) (*model.Message, error)
}

type GetServerSignedMessageUseCase struct {
	repository serverMessageFinder
}

func NewGetServerSignedMessageUseCase(repository serverMessageFinder) *GetServerSignedMessageUseCase {
	return &GetServerSignedMessageUseCase{
		repository: repository,
	}
}

func (uc *GetServerSignedMessageUseCase) Execute(ctx context.Context, id string) (*model.Message, error) {
	return uc.repository.FindByID(ctx, id)
}
