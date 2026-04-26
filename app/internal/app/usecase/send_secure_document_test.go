package usecase

import (
	"context"
	"strings"
	"testing"

	"electronic-digital-signature/internal/domain/model"
	"electronic-digital-signature/internal/infra/encryption"
)

func TestSendDocumentUseCaseSignsEncryptsAndSendsPackage(t *testing.T) {
	repository := &fakeSendDocumentRepository{
		document: model.Document{
			ID:               "document-id",
			StoredPath:       "stored/document.docx",
			OriginalFileName: "contract.docx",
			MimeType:         "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		},
	}
	storage := &fakeSecureDocumentStorage{
		contentByPath: map[string][]byte{
			"stored/document.docx": []byte("signed document bytes"),
		},
	}
	signer := &fakeDocumentSigner{}
	encryptor := &fakeDocumentEncryptor{storage: storage}
	mailer := &fakeMailer{}

	result, err := NewSendDocumentUseCase(
		repository,
		storage,
		signer,
		[]byte("server-private-key"),
		encryptor,
		mailer,
	).Execute(context.Background(), SendDocumentInput{
		DocumentID:     "document-id",
		RecipientEmail: "recipient@example.com",
		SentByUserID:   "sender-user-id",
	})
	if err != nil {
		t.Fatalf("send document: %v", err)
	}

	if result.PackageID != "document-id_encrypted_package" {
		t.Fatalf("expected package id, got %q", result.PackageID)
	}
	if result.SendStatus != DocumentSendStatusSent {
		t.Fatalf("expected send status sent, got %q", result.SendStatus)
	}
	if string(signer.signedMessage) != "signed document bytes" {
		t.Fatalf("expected signer to receive document content, got %q", signer.signedMessage)
	}
	if string(encryptor.document.Signature) != "signature" {
		t.Fatalf("expected encryptor to receive signed document, got signature %q", encryptor.document.Signature)
	}
	if repository.document.EncryptedPath != "stored/document-id_encrypted_package.json" {
		t.Fatalf("expected encrypted path to be saved, got %q", repository.document.EncryptedPath)
	}
	if repository.document.SendStatus != DocumentSendStatusSent {
		t.Fatalf("expected saved send status, got %q", repository.document.SendStatus)
	}
	if repository.document.LastSentByUserID != "sender-user-id" {
		t.Fatalf("expected saved last sent by user id, got %q", repository.document.LastSentByUserID)
	}
	if len(mailer.attachments) != 1 {
		t.Fatalf("expected package attachment, got %d", len(mailer.attachments))
	}
	attachmentContent := string(mailer.attachments[0].Content)
	if strings.Contains(attachmentContent, "server-private-key") {
		t.Fatalf("private key leaked into attachment: %q", attachmentContent)
	}
	if strings.Contains(mailer.body, "server-private-key") {
		t.Fatalf("private key leaked into email body: %q", mailer.body)
	}
}

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

type fakeSendDocumentRepository struct {
	document model.Document
}

func (r *fakeSendDocumentRepository) FindByID(_ context.Context, id string) (*model.Document, error) {
	if r.document.ID != id {
		return nil, errFakeNotFound
	}

	return &r.document, nil
}

func (r *fakeSendDocumentRepository) Update(_ context.Context, document *model.Document) error {
	r.document = *document
	return nil
}

type fakeSecureDocumentStorage struct {
	contentByPath map[string][]byte
}

func (s *fakeSecureDocumentStorage) Read(_ context.Context, path string) ([]byte, error) {
	return s.contentByPath[path], nil
}

type fakeDocumentSigner struct {
	signedMessage []byte
	privateKey    []byte
}

func (s *fakeDocumentSigner) Hash(message []byte) []byte {
	return []byte("hash")
}

func (s *fakeDocumentSigner) Sign(message []byte, privateKey []byte) ([]byte, error) {
	s.signedMessage = append([]byte(nil), message...)
	s.privateKey = append([]byte(nil), privateKey...)
	return []byte("signature"), nil
}

type fakeDocumentEncryptor struct {
	storage  *fakeSecureDocumentStorage
	document model.Document
	content  []byte
}

func (e *fakeDocumentEncryptor) EncryptAndSave(_ context.Context, document model.Document, content []byte) (encryption.EncryptedPackage, string, error) {
	e.document = document
	e.content = append([]byte(nil), content...)
	path := "stored/" + document.ID + "_encrypted_package.json"
	e.storage.contentByPath[path] = []byte(`{"document_id":"` + document.ID + `"}`)
	return encryption.EncryptedPackage{DocumentID: document.ID}, path, nil
}

type fakeNotFoundError string

func (e fakeNotFoundError) Error() string {
	return string(e)
}

const errFakeNotFound = fakeNotFoundError("document not found")
