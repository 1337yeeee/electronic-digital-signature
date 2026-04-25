package usecase

import (
	"context"
	"strings"
	"testing"

	"electronic-digital-signature/internal/domain/model"
)

func TestSendSecureDocumentSendsEncryptedPackageAttachment(t *testing.T) {
	mailer := &fakeMailer{}
	document := model.Document{
		ID:               "document-id",
		OriginalFileName: "contract.docx",
	}
	encryptedPackage := []byte(`{"document_id":"document-id"}`)

	err := SendSecureDocument(context.Background(), mailer, SendSecureDocumentInput{
		Document:         document,
		To:               []string{"recipient@example.com"},
		Subject:          "Encrypted document",
		EncryptedPackage: encryptedPackage,
	})
	if err != nil {
		t.Fatalf("send secure document: %v", err)
	}

	if len(mailer.attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(mailer.attachments))
	}
	attachment := mailer.attachments[0]
	if attachment.FileName != "document-id_encrypted_package.json" {
		t.Fatalf("expected default attachment name, got %q", attachment.FileName)
	}
	if attachment.ContentType != "application/json" {
		t.Fatalf("expected attachment content type application/json, got %q", attachment.ContentType)
	}
	if string(attachment.Content) != string(encryptedPackage) {
		t.Fatalf("expected attachment content %q, got %q", encryptedPackage, attachment.Content)
	}
	if mailer.body == "" {
		t.Fatal("expected email body")
	}
	if !strings.Contains(mailer.body, "Document ID: document-id") {
		t.Fatalf("expected document_id in body, got %q", mailer.body)
	}
	if !strings.Contains(mailer.body, "Encryption algorithm: AES-256-GCM") {
		t.Fatalf("expected encryption algorithm in body, got %q", mailer.body)
	}
}

func TestSendSecureDocumentRejectsEmptyEncryptedPackage(t *testing.T) {
	err := SendSecureDocument(context.Background(), &fakeMailer{}, SendSecureDocumentInput{
		Document: model.Document{ID: "document-id"},
		To:       []string{"recipient@example.com"},
		Subject:  "Encrypted document",
	})
	if err == nil {
		t.Fatal("expected empty encrypted package to fail")
	}
}

type fakeMailer struct {
	to          []string
	subject     string
	body        string
	attachments []EmailAttachment
}

func (m *fakeMailer) SendEmail(_ context.Context, to []string, subject, body string, attachments []EmailAttachment) error {
	m.to = to
	m.subject = subject
	m.body = body
	m.attachments = attachments
	return nil
}
