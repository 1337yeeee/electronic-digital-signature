package usecase

import (
	"context"
	"electronic-digital-signature/internal/domain/model"
)

type mailer interface {
	SendEmail(ctx context.Context, to []string, subject, content string) error
}

func SendSecureDocument(ctx context.Context, document model.Document, mailer mailer, to []string, subject string, privateKey, publicKer []byte) error {
	//TODO SendSecureDocument
	content := document.Message.Message + string(privateKey) + string(publicKer)
	return mailer.SendEmail(ctx, to, subject, content)
}
