package routes

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"electronic-digital-signature/internal/app/container"
	"electronic-digital-signature/internal/app/dto"
	"electronic-digital-signature/internal/app/handler"
	"electronic-digital-signature/internal/app/usecase"
	"electronic-digital-signature/internal/domain/model"
	infraauth "electronic-digital-signature/internal/infra/auth"
	"electronic-digital-signature/internal/infra/crypto"
	"electronic-digital-signature/internal/infra/docx"
	"electronic-digital-signature/internal/infra/encryption"
	"electronic-digital-signature/internal/infra/keys"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestHealthRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := SetupRouter(nil)

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	expectedBody := `{"data":{"status":"ok"},"success":true}`
	if response.Body.String() != expectedBody {
		t.Fatalf("expected body %s, got %s", expectedBody, response.Body.String())
	}
}

func TestUploadDocumentRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, publicKey := generateECDSAKeyPairPEM(t)
	authSession := newTestAuthSession(t)
	router, documentRepository, documentStorage := setupProtectedRouterWithDocumentHandler(privateKey, authSession)

	response := performMultipartDocumentUploadWithToken(t, router, "contract.docx", authSession.token)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.Code)
	}

	var envelope struct {
		Success bool                       `json:"success"`
		Data    dto.UploadDocumentResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if !envelope.Success {
		t.Fatal("expected success response")
	}
	body := envelope.Data

	if body.DocumentID != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("expected document_id from generator, got %q", body.DocumentID)
	}
	if body.OwnerUserID != authSession.user.ID {
		t.Fatalf("expected owner user id %q, got %q", authSession.user.ID, body.OwnerUserID)
	}
	if body.SignedByUserID != authSession.user.ID {
		t.Fatalf("expected signed by user id %q, got %q", authSession.user.ID, body.SignedByUserID)
	}
	if body.OwnerEmail != authSession.user.Email {
		t.Fatalf("expected owner email, got %q", body.OwnerEmail)
	}
	if body.RecipientEmail != "recipient@example.com" {
		t.Fatalf("expected recipient email, got %q", body.RecipientEmail)
	}
	if body.OriginalFileName != "contract.docx" {
		t.Fatalf("expected original file name, got %q", body.OriginalFileName)
	}
	if body.StoredPath != "stored/contract.docx" {
		t.Fatalf("expected stored path, got %q", body.StoredPath)
	}
	if len(documentRepository.documents) != 1 {
		t.Fatalf("expected 1 saved document, got %d", len(documentRepository.documents))
	}
	if documentRepository.documents[0].OwnerUserID != authSession.user.ID {
		t.Fatalf("expected saved owner user id %q, got %q", authSession.user.ID, documentRepository.documents[0].OwnerUserID)
	}
	if documentRepository.documents[0].SignedByUserID != authSession.user.ID {
		t.Fatalf("expected saved signed by user id %q, got %q", authSession.user.ID, documentRepository.documents[0].SignedByUserID)
	}
	if len(documentRepository.documents[0].Hash) == 0 {
		t.Fatal("expected saved document hash")
	}
	if len(documentRepository.documents[0].Signature) == 0 {
		t.Fatal("expected saved document signature")
	}
	documentXML := readDocxDocumentXML(t, documentStorage.content)
	if !strings.Contains(documentXML, "Document UUID: 00000000-0000-4000-8000-000000000001") {
		t.Fatalf("expected document UUID metadata in document.xml, got %q", documentXML)
	}
	if !strings.Contains(documentXML, "Date: ") {
		t.Fatalf("expected date metadata in document.xml, got %q", documentXML)
	}

	assertDocumentSignature(t, documentStorage.content, documentRepository.documents[0], publicKey)
	tamperedContent := append([]byte(nil), documentStorage.content...)
	tamperedContent[len(tamperedContent)-1] ^= 0xff
	if err := crypto.NewECDSASHA256Provider().Verify(tamperedContent, documentRepository.documents[0].Signature, publicKey); err == nil {
		t.Fatal("expected tampered document verification to fail")
	}
}

func TestUploadDocumentRouteRejectsNonDocxFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, _ := generateECDSAKeyPairPEM(t)
	authSession := newTestAuthSession(t)
	router, _, _ := setupProtectedRouterWithDocumentHandler(privateKey, authSession)

	response := performMultipartDocumentUploadWithToken(t, router, "contract.txt", authSession.token)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}

	var body dto.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error.Code != "invalid_document_type" {
		t.Fatalf("unexpected error code: %q", body.Error.Code)
	}
	if body.Error.Message != "Document file must have .docx extension." {
		t.Fatalf("unexpected error message: %q", body.Error.Message)
	}
}

func TestUploadDocumentRouteRejectsUnsupportedMIMEType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, _ := generateECDSAKeyPairPEM(t)
	authSession := newTestAuthSession(t)
	router, _, _ := setupProtectedRouterWithDocumentHandler(privateKey, authSession)

	response := performMultipartDocumentUploadWithOptionsAndToken(t, router, "contract.docx", minimalDocx(t), "text/plain", authSession.token)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}

	var body dto.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error.Code != "invalid_document_type" {
		t.Fatalf("unexpected error code: %q", body.Error.Code)
	}
	if body.Error.Message != "Document MIME type is not supported." {
		t.Fatalf("unexpected error message: %q", body.Error.Message)
	}
}

func TestUploadDocumentRouteRejectsTooLargeDocument(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, _ := generateECDSAKeyPairPEM(t)
	authSession := newTestAuthSession(t)
	router, _, _ := setupProtectedRouterWithDocumentHandler(privateKey, authSession)

	tooLargeContent := bytes.Repeat([]byte("a"), usecase.MaxUploadDocumentSizeBytes+1)
	response := performMultipartDocumentUploadWithOptionsAndToken(t, router, "contract.docx", tooLargeContent, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", authSession.token)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}

	var body dto.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error.Code != "document_too_large" {
		t.Fatalf("unexpected error code: %q", body.Error.Code)
	}
}

func TestRegisterUserRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userRepository := &fakeUserRepository{}
	router := setupRouterWithUserHandler(userRepository)
	_, publicKey := generateECDSAKeyPairPEM(t)

	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/users/register", dto.RegisterUserRequest{
		Email:        "user@example.com",
		Name:         "Lab User",
		Password:     "secret-password",
		PublicKeyPEM: string(publicKey),
	})

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}

	var envelope struct {
		Success bool             `json:"success"`
		Data    dto.UserResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if !envelope.Success {
		t.Fatal("expected success response")
	}
	if envelope.Data.Email != "user@example.com" {
		t.Fatalf("expected user email, got %q", envelope.Data.Email)
	}
	if envelope.Data.Name != "Lab User" {
		t.Fatalf("expected user name, got %q", envelope.Data.Name)
	}
	if envelope.Data.PublicKeyPEM != strings.TrimSpace(string(publicKey)) {
		t.Fatalf("expected public key pem, got %q", envelope.Data.PublicKeyPEM)
	}
	if envelope.Data.UpdatedAt == "" {
		t.Fatal("expected updated_at")
	}
	if len(userRepository.users) != 1 {
		t.Fatalf("expected 1 saved user, got %d", len(userRepository.users))
	}
	if len(userRepository.keyHistory) != 1 {
		t.Fatalf("expected 1 key history entry, got %d", len(userRepository.keyHistory))
	}
	if userRepository.users[0].PasswordHash == "secret-password" || userRepository.users[0].PasswordHash == "" {
		t.Fatal("expected password hash to be stored instead of raw password")
	}
}

func TestRegisterUserRouteRejectsDuplicateEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userRepository := &fakeUserRepository{
		users: []model.User{
			{
				ID:           "user-id",
				Email:        "user@example.com",
				Name:         "Existing User",
				PasswordHash: "hash",
			},
		},
	}
	router := setupRouterWithUserHandler(userRepository)

	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/users/register", dto.RegisterUserRequest{
		Email:    "user@example.com",
		Name:     "Another User",
		Password: "secret-password",
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}

	var body dto.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error.Code != "email_already_exists" {
		t.Fatalf("unexpected error code: %q", body.Error.Code)
	}
}

func TestRegisterUserRouteRejectsInvalidPublicKeyPEM(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupRouterWithUserHandler(&fakeUserRepository{})

	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/users/register", dto.RegisterUserRequest{
		Email:        "user@example.com",
		Name:         "Lab User",
		Password:     "secret-password",
		PublicKeyPEM: "not-a-pem-key",
	})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}

	var body dto.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error.Code != "invalid_public_key" {
		t.Fatalf("unexpected error code: %q", body.Error.Code)
	}
}

func TestGetUserRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userRepository := &fakeUserRepository{
		users: []model.User{
			{
				ID:           "user-id",
				Email:        "user@example.com",
				Name:         "Lab User",
				PasswordHash: "hash",
				PublicKeyPEM: "pem",
				CreatedAt:    time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
				UpdatedAt:    time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	router := setupRouterWithUserHandler(userRepository)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/users/user-id", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var envelope struct {
		Success bool             `json:"success"`
		Data    dto.UserResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if !envelope.Success {
		t.Fatal("expected success response")
	}
	if envelope.Data.ID != "user-id" {
		t.Fatalf("expected user id, got %q", envelope.Data.ID)
	}
	if envelope.Data.UpdatedAt == "" {
		t.Fatal("expected updated_at")
	}
}

func TestUpdateMyPublicKeyRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldPrivateKey, oldPublicKey := generateECDSAKeyPairPEM(t)
	_ = oldPrivateKey
	newPrivateKey, newPublicKey := generateECDSAKeyPairPEM(t)
	_ = newPrivateKey
	authSession := newTestAuthSessionWithPublicKey(t, string(oldPublicKey))
	router := setupProtectedRouterWithUserHandler(authSession.userRepository, authSession)

	response := performJSONRequestWithToken(t, router, http.MethodPut, "/api/v1/users/me/public-key", dto.UpdateMyPublicKeyRequest{
		PublicKeyPEM: string(newPublicKey),
	}, authSession.token)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}

	var envelope struct {
		Success bool             `json:"success"`
		Data    dto.UserResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if !envelope.Success {
		t.Fatal("expected success response")
	}
	if envelope.Data.PublicKeyPEM != strings.TrimSpace(string(newPublicKey)) {
		t.Fatalf("expected updated public key, got %q", envelope.Data.PublicKeyPEM)
	}
	if envelope.Data.UpdatedAt == "" {
		t.Fatal("expected updated_at")
	}
	if authSession.userRepository.users[0].PublicKeyPEM != strings.TrimSpace(string(newPublicKey)) {
		t.Fatalf("expected saved public key to be updated, got %q", authSession.userRepository.users[0].PublicKeyPEM)
	}
	if len(authSession.userRepository.keyHistory) != 2 {
		t.Fatalf("expected 2 key history entries, got %d", len(authSession.userRepository.keyHistory))
	}
}

func TestUpdateMyPublicKeyRouteRejectsInvalidPublicKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authSession := newTestAuthSession(t)
	router := setupProtectedRouterWithUserHandler(authSession.userRepository, authSession)

	response := performJSONRequestWithToken(t, router, http.MethodPut, "/api/v1/users/me/public-key", dto.UpdateMyPublicKeyRequest{
		PublicKeyPEM: "not-a-pem-key",
	}, authSession.token)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}

	var body dto.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error.Code != "invalid_public_key" {
		t.Fatalf("unexpected error code: %q", body.Error.Code)
	}
}

func TestLoginRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate password hash: %v", err)
	}

	userRepository := &fakeUserRepository{
		users: []model.User{
			{
				ID:           "user-id",
				Email:        "user@example.com",
				Name:         "Lab User",
				PasswordHash: string(passwordHash),
				CreatedAt:    time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	router, _ := setupRouterWithAuth(userRepository)

	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/auth/login", dto.LoginRequest{
		Email:    "user@example.com",
		Password: "secret-password",
	})

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}

	var envelope struct {
		Success bool              `json:"success"`
		Data    dto.LoginResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if !envelope.Success {
		t.Fatal("expected success response")
	}
	if envelope.Data.AccessToken == "" {
		t.Fatal("expected access token")
	}
	if envelope.Data.TokenType != "Bearer" {
		t.Fatalf("expected Bearer token type, got %q", envelope.Data.TokenType)
	}
	if envelope.Data.User.ID != "user-id" {
		t.Fatalf("expected user id, got %q", envelope.Data.User.ID)
	}
}

func TestLoginRouteRejectsInvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate password hash: %v", err)
	}

	userRepository := &fakeUserRepository{
		users: []model.User{
			{
				ID:           "user-id",
				Email:        "user@example.com",
				Name:         "Lab User",
				PasswordHash: string(passwordHash),
			},
		},
	}
	router, _ := setupRouterWithAuth(userRepository)

	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/auth/login", dto.LoginRequest{
		Email:    "user@example.com",
		Password: "wrong-password",
	})

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}

	var body dto.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error.Code != "invalid_credentials" {
		t.Fatalf("unexpected error code: %q", body.Error.Code)
	}
}

func TestAuthMeRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate password hash: %v", err)
	}

	userRepository := &fakeUserRepository{
		users: []model.User{
			{
				ID:           "user-id",
				Email:        "user@example.com",
				Name:         "Lab User",
				PasswordHash: string(passwordHash),
				CreatedAt:    time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	router, jwtManager := setupRouterWithAuth(userRepository)
	accessToken, _, err := jwtManager.Generate("user-id", "user@example.com")
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}

	var envelope struct {
		Success bool             `json:"success"`
		Data    dto.UserResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if !envelope.Success {
		t.Fatal("expected success response")
	}
	if envelope.Data.ID != "user-id" {
		t.Fatalf("expected user id, got %q", envelope.Data.ID)
	}
}

func TestAuthMeRouteRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, _ := setupRouterWithAuth(&fakeUserRepository{})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}

	var body dto.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error.Code != "unauthorized" {
		t.Fatalf("unexpected error code: %q", body.Error.Code)
	}
}

func TestSendDocumentRouteSendsEncryptedPackageAndStoresStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authSession := newTestAuthSession(t)
	documentRepository := &fakeDocumentRepository{
		documents: []model.Document{
			{
				ID:               "document-id",
				OwnerUserID:      authSession.user.ID,
				SignedByUserID:   authSession.user.ID,
				RecipientEmail:   "old-recipient@example.com",
				OriginalFileName: "contract.docx",
				EncryptedPath:    "stored/document-id_encrypted_package.json",
			},
		},
	}
	documentStorage := &fakeDocumentStorage{
		encryptedPackageContent: []byte(`{"document_id":"document-id"}`),
	}
	mailer := &fakeMailer{}

	router := SetupRouter(&container.AppContainer{
		AuthMiddleware: newAuthMiddlewareForSession(authSession),
		DocumentHandler: handler.NewDocumentHandler(
			nil,
			usecase.NewSendDocumentUseCase(documentRepository, documentStorage, nil, nil, nil, mailer),
			nil,
			nil,
		),
	})

	response := performJSONRequestWithToken(t, router, http.MethodPost, "/api/v1/documents/document-id/send", dto.SendDocumentRequest{
		Email: "recipient@example.com",
	}, authSession.token)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}

	var envelope struct {
		Success bool                     `json:"success"`
		Data    dto.SendDocumentResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if !envelope.Success {
		t.Fatal("expected success response")
	}
	body := envelope.Data
	if body.DocumentID != "document-id" {
		t.Fatalf("expected document_id, got %q", body.DocumentID)
	}
	if body.PackageID != "document-id_encrypted_package" {
		t.Fatalf("expected package_id, got %q", body.PackageID)
	}
	if body.RecipientEmail != "recipient@example.com" {
		t.Fatalf("expected recipient email, got %q", body.RecipientEmail)
	}
	if body.SendStatus != usecase.DocumentSendStatusSent {
		t.Fatalf("expected send status sent, got %q", body.SendStatus)
	}
	if body.OwnerUserID != authSession.user.ID {
		t.Fatalf("expected owner_user_id %q, got %q", authSession.user.ID, body.OwnerUserID)
	}
	if body.SignedByUserID != authSession.user.ID {
		t.Fatalf("expected signed_by_user_id %q, got %q", authSession.user.ID, body.SignedByUserID)
	}
	if body.SentByUserID != authSession.user.ID {
		t.Fatalf("expected sent_by_user_id %q, got %q", authSession.user.ID, body.SentByUserID)
	}
	if body.SentAt == "" {
		t.Fatal("expected sent_at")
	}

	if len(mailer.attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(mailer.attachments))
	}
	if mailer.attachments[0].FileName != "document-id_encrypted_package.json" {
		t.Fatalf("expected encrypted package attachment, got %q", mailer.attachments[0].FileName)
	}
	if string(mailer.attachments[0].Content) != `{"document_id":"document-id"}` {
		t.Fatalf("unexpected attachment content %q", string(mailer.attachments[0].Content))
	}
	if !strings.Contains(mailer.body, "Document ID: document-id") {
		t.Fatalf("expected document id in email body, got %q", mailer.body)
	}
	if !strings.Contains(mailer.body, "Encryption algorithm: AES-256-GCM") {
		t.Fatalf("expected algorithm in email body, got %q", mailer.body)
	}

	savedDocument := documentRepository.documents[0]
	if savedDocument.SendStatus != usecase.DocumentSendStatusSent {
		t.Fatalf("expected saved status sent, got %q", savedDocument.SendStatus)
	}
	if savedDocument.LastSentByUserID != authSession.user.ID {
		t.Fatalf("expected saved last_sent_by_user_id %q, got %q", authSession.user.ID, savedDocument.LastSentByUserID)
	}
	if savedDocument.LastSentToEmail != "recipient@example.com" {
		t.Fatalf("expected saved sent email, got %q", savedDocument.LastSentToEmail)
	}
	if savedDocument.SentAt == nil {
		t.Fatal("expected saved sent_at")
	}
}

func TestSendDocumentRouteReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authSession := newTestAuthSession(t)
	router := SetupRouter(&container.AppContainer{
		AuthMiddleware: newAuthMiddlewareForSession(authSession),
		DocumentHandler: handler.NewDocumentHandler(
			nil,
			usecase.NewSendDocumentUseCase(&fakeDocumentRepository{}, &fakeDocumentStorage{}, nil, nil, nil, &fakeMailer{}),
			nil,
			nil,
		),
	})

	response := performJSONRequestWithToken(t, router, http.MethodPost, "/api/v1/documents/missing/send", dto.SendDocumentRequest{
		Email: "recipient@example.com",
	}, authSession.token)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestSendDocumentRouteReturnsForbiddenForForeignOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authSession := newTestAuthSession(t)
	documentRepository := &fakeDocumentRepository{
		documents: []model.Document{
			{
				ID:               "document-id",
				OwnerUserID:      "different-owner-id",
				SignedByUserID:   "different-owner-id",
				RecipientEmail:   "old-recipient@example.com",
				OriginalFileName: "contract.docx",
				EncryptedPath:    "stored/document-id_encrypted_package.json",
			},
		},
	}
	documentStorage := &fakeDocumentStorage{
		encryptedPackageContent: []byte(`{"document_id":"document-id"}`),
	}
	mailer := &fakeMailer{}

	router := SetupRouter(&container.AppContainer{
		AuthMiddleware: newAuthMiddlewareForSession(authSession),
		DocumentHandler: handler.NewDocumentHandler(
			nil,
			usecase.NewSendDocumentUseCase(documentRepository, documentStorage, nil, nil, nil, mailer),
			nil,
			nil,
		),
	})

	response := performJSONRequestWithToken(t, router, http.MethodPost, "/api/v1/documents/document-id/send", dto.SendDocumentRequest{
		Email: "recipient@example.com",
	}, authSession.token)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, response.Code, response.Body.String())
	}

	var body dto.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error.Code != "forbidden" {
		t.Fatalf("unexpected error code: %q", body.Error.Code)
	}
	if len(mailer.attachments) != 0 {
		t.Fatalf("expected no mail to be sent, got %d attachments", len(mailer.attachments))
	}
}

func TestGetDocumentAuditRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authSession := newTestAuthSession(t)
	sentAt := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	documentRepository := &fakeDocumentRepository{
		documents: []model.Document{
			{
				ID:               "document-id",
				OwnerUserID:      authSession.user.ID,
				SignedByUserID:   authSession.user.ID,
				LastSentByUserID: authSession.user.ID,
				OwnerEmail:       authSession.user.Email,
				RecipientEmail:   "recipient@example.com",
				OriginalFileName: "contract.docx",
				MimeType:         "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
				SendStatus:       usecase.DocumentSendStatusSent,
				CreatedAt:        time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
				SignedAt:         time.Date(2026, 4, 26, 10, 1, 0, 0, time.UTC),
				SentAt:           &sentAt,
			},
		},
	}

	router := SetupRouter(&container.AppContainer{
		AuthMiddleware: newAuthMiddlewareForSession(authSession),
		DocumentHandler: handler.NewDocumentHandler(
			nil,
			nil,
			usecase.NewGetDocumentAuditUseCase(documentRepository),
			nil,
		),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/documents/document-id/audit", nil)
	request.Header.Set("Authorization", "Bearer "+authSession.token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}

	var envelope struct {
		Success bool                      `json:"success"`
		Data    dto.DocumentAuditResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if !envelope.Success {
		t.Fatal("expected success response")
	}
	if envelope.Data.OwnerUserID != authSession.user.ID {
		t.Fatalf("expected owner_user_id %q, got %q", authSession.user.ID, envelope.Data.OwnerUserID)
	}
	if envelope.Data.SignedByUserID != authSession.user.ID {
		t.Fatalf("expected signed_by_user_id %q, got %q", authSession.user.ID, envelope.Data.SignedByUserID)
	}
	if envelope.Data.SentByUserID != authSession.user.ID {
		t.Fatalf("expected sent_by_user_id %q, got %q", authSession.user.ID, envelope.Data.SentByUserID)
	}
}

func TestGetDocumentAuditRouteReturnsForbiddenForForeignOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authSession := newTestAuthSession(t)
	documentRepository := &fakeDocumentRepository{
		documents: []model.Document{
			{
				ID:             "document-id",
				OwnerUserID:    "different-owner-id",
				SignedByUserID: "different-owner-id",
			},
		},
	}

	router := SetupRouter(&container.AppContainer{
		AuthMiddleware: newAuthMiddlewareForSession(authSession),
		DocumentHandler: handler.NewDocumentHandler(
			nil,
			nil,
			usecase.NewGetDocumentAuditUseCase(documentRepository),
			nil,
		),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/documents/document-id/audit", nil)
	request.Header.Set("Authorization", "Bearer "+authSession.token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, response.Code, response.Body.String())
	}
}

func TestVerifyDecryptPackageRouteAcceptsPackageJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, publicKey := generateECDSAKeyPairPEM(t)
	documentContent := []byte("decrypted document bytes")
	packageContent := encryptedPackageContent(t, privateKey, documentContent)
	router := setupRouterWithVerifyDecryptPackageHandler(publicKey)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents/verify-decrypt", bytes.NewReader(packageContent))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}

	var envelope struct {
		Success bool                             `json:"success"`
		Data    dto.VerifyDecryptPackageResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if !envelope.Success {
		t.Fatal("expected success response")
	}
	body := envelope.Data
	if !body.Valid {
		t.Fatal("expected package to be valid")
	}
	if body.Metadata.DocumentID != "document-id" {
		t.Fatalf("expected document metadata, got %+v", body.Metadata)
	}
	if body.Metadata.EncryptionAlgorithm != encryption.AESGCMAlgorithm {
		t.Fatalf("expected encryption algorithm %q, got %q", encryption.AESGCMAlgorithm, body.Metadata.EncryptionAlgorithm)
	}

	decryptedDocument, err := base64.StdEncoding.DecodeString(body.DecryptedDocumentBase64)
	if err != nil {
		t.Fatalf("decode decrypted document: %v", err)
	}
	if string(decryptedDocument) != string(documentContent) {
		t.Fatalf("expected decrypted document %q, got %q", documentContent, decryptedDocument)
	}
}

func TestVerifyDecryptPackageRouteAcceptsPackageFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, publicKey := generateECDSAKeyPairPEM(t)
	documentContent := []byte("docx bytes from multipart package")
	packageContent := encryptedPackageContent(t, privateKey, documentContent)
	router := setupRouterWithVerifyDecryptPackageHandler(publicKey)

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	fileWriter, err := writer.CreateFormFile("package", "document-id_encrypted_package.json")
	if err != nil {
		t.Fatalf("create package file field: %v", err)
	}
	if _, err := fileWriter.Write(packageContent); err != nil {
		t.Fatalf("write package file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents/verify-decrypt", &requestBody)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}

	var envelope struct {
		Success bool                             `json:"success"`
		Data    dto.VerifyDecryptPackageResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if !envelope.Success {
		t.Fatal("expected success response")
	}
	body := envelope.Data
	if !body.Valid {
		t.Fatal("expected package to be valid")
	}
}

func TestVerifyDecryptPackageRouteReturnsErrorForWrongServerPublicKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, _ := generateECDSAKeyPairPEM(t)
	_, wrongPublicKey := generateECDSAKeyPairPEM(t)
	packageContent := encryptedPackageContent(t, privateKey, []byte("document bytes"))
	router := setupRouterWithVerifyDecryptPackageHandler(wrongPublicKey)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents/verify-decrypt", bytes.NewReader(packageContent))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, response.Code, response.Body.String())
	}

	var body dto.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error.Code != "invalid_signature" {
		t.Fatalf("expected invalid_signature error code, got %q", body.Error.Code)
	}
	if body.Error.Message != "Package signature is invalid." {
		t.Fatalf("unexpected error message: %q", body.Error.Message)
	}
}

func TestVerifyDecryptPackageRouteReturnsErrorForModifiedSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, publicKey := generateECDSAKeyPairPEM(t)
	packageContent := tamperedPackageSignatureContent(t, encryptedPackageContent(t, privateKey, []byte("document bytes")))
	router := setupRouterWithVerifyDecryptPackageHandler(publicKey)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents/verify-decrypt", bytes.NewReader(packageContent))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, response.Code, response.Body.String())
	}

	var body dto.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error.Code != "invalid_signature" {
		t.Fatalf("expected invalid_signature error code, got %q", body.Error.Code)
	}
	if body.Error.Message != "Package signature is invalid." {
		t.Fatalf("unexpected error message: %q", body.Error.Message)
	}
}

func TestVerifyDecryptPackageRouteReturnsClearErrorForCorruptedPackage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, publicKey := generateECDSAKeyPairPEM(t)
	packageContent := tamperedPackageCiphertextContent(t, encryptedPackageContent(t, privateKey, []byte("document bytes")))
	router := setupRouterWithVerifyDecryptPackageHandler(publicKey)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents/verify-decrypt", bytes.NewReader(packageContent))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, response.Code, response.Body.String())
	}

	var body dto.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error.Code != "invalid_package" {
		t.Fatalf("expected invalid_package error code, got %q", body.Error.Code)
	}
	if body.Error.Message != "Encrypted package is invalid." {
		t.Fatalf("unexpected error message: %q", body.Error.Message)
	}
}

func TestServerPublicKeyRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	publicKey := []byte("-----BEGIN PUBLIC KEY-----\ntest\n-----END PUBLIC KEY-----\n")
	router := setupRouterWithSignatureHandler(keys.ServerKeyPair{PublicKey: publicKey})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/server/public-key", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body dto.ServerPublicKeyResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.Algorithm != crypto.ECDSASHA256Algorithm {
		t.Fatalf("expected algorithm %q, got %q", crypto.ECDSASHA256Algorithm, body.Algorithm)
	}
	if body.PublicKeyPEM != string(publicKey) {
		t.Fatalf("expected public key %q, got %q", string(publicKey), body.PublicKeyPEM)
	}
}

func TestServerPublicKeyRouteReturnsErrorWhenKeyIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupRouterWithSignatureHandler(keys.ServerKeyPair{})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/server/public-key", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["error"] != "server public key is not loaded" {
		t.Fatalf("unexpected error: %q", body["error"])
	}
}

func TestVerifyClientSignatureRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := crypto.NewECDSASHA256Provider()
	privateKey, publicKey := generateECDSAKeyPairPEM(t)
	message := "client signed message"

	signature, err := provider.Sign([]byte(message), privateKey)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}

	requestBody := dto.VerifyClientSignatureRequest{
		Message:         message,
		SignatureBase64: base64.StdEncoding.EncodeToString(signature),
		PublicKey:       string(publicKey),
	}
	router := setupRouterWithSignatureHandler(keys.ServerKeyPair{})
	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/signatures/verify", requestBody)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body dto.VerifyClientSignatureResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if !body.Valid {
		t.Fatalf("expected valid signature, got error %q", body.Error)
	}
	if body.SignerType != "user" {
		t.Fatalf("expected signer_type user, got %q", body.SignerType)
	}
}

func TestVerifyClientSignatureRouteReturnsInvalidForModifiedMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := crypto.NewECDSASHA256Provider()
	privateKey, publicKey := generateECDSAKeyPairPEM(t)

	signature, err := provider.Sign([]byte("original message"), privateKey)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}

	requestBody := dto.VerifyClientSignatureRequest{
		Message:         "modified message",
		SignatureBase64: base64.StdEncoding.EncodeToString(signature),
		PublicKey:       string(publicKey),
	}
	router := setupRouterWithSignatureHandler(keys.ServerKeyPair{})
	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/signatures/verify", requestBody)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body dto.VerifyClientSignatureResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Valid {
		t.Fatal("expected modified message signature to be invalid")
	}
	if body.Error == "" {
		t.Fatal("expected verification error")
	}
	if body.SignerType != "user" {
		t.Fatalf("expected signer_type user, got %q", body.SignerType)
	}
}

func TestVerifyCurrentUserSignatureRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := crypto.NewECDSASHA256Provider()
	privateKey, publicKey := generateECDSAKeyPairPEM(t)
	authSession := newTestAuthSessionWithPublicKey(t, string(publicKey))
	message := "user signed message"

	signature, err := provider.Sign([]byte(message), privateKey)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}

	router := setupProtectedRouterWithSignatureHandler(keys.ServerKeyPair{}, authSession)
	response := performJSONRequestWithToken(t, router, http.MethodPost, "/api/v1/users/me/signatures/verify", dto.VerifyUserSignatureRequest{
		Message:         message,
		SignatureBase64: base64.StdEncoding.EncodeToString(signature),
	}, authSession.token)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body dto.VerifyClientSignatureResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if !body.Valid {
		t.Fatalf("expected valid signature, got error %q", body.Error)
	}
	if body.SignerType != "user" {
		t.Fatalf("expected signer_type user, got %q", body.SignerType)
	}
	if body.SignerUserID != authSession.user.ID {
		t.Fatalf("expected signer_user_id %q, got %q", authSession.user.ID, body.SignerUserID)
	}
}

func TestVerifyCurrentUserSignatureRouteReturnsInvalidForWrongSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	provider := crypto.NewECDSASHA256Provider()
	privateKey, publicKey := generateECDSAKeyPairPEM(t)
	authSession := newTestAuthSessionWithPublicKey(t, string(publicKey))

	signature, err := provider.Sign([]byte("original message"), privateKey)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}

	router := setupProtectedRouterWithSignatureHandler(keys.ServerKeyPair{}, authSession)
	response := performJSONRequestWithToken(t, router, http.MethodPost, "/api/v1/users/me/signatures/verify", dto.VerifyUserSignatureRequest{
		Message:         "tampered message",
		SignatureBase64: base64.StdEncoding.EncodeToString(signature),
	}, authSession.token)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body dto.VerifyClientSignatureResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Valid {
		t.Fatal("expected invalid signature")
	}
	if body.SignerType != "user" {
		t.Fatalf("expected signer_type user, got %q", body.SignerType)
	}
	if body.SignerUserID != authSession.user.ID {
		t.Fatalf("expected signer_user_id %q, got %q", authSession.user.ID, body.SignerUserID)
	}
}

func TestVerifyCurrentUserSignatureRouteRejectsMissingRegisteredPublicKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authSession := newTestAuthSession(t)
	router := setupProtectedRouterWithSignatureHandler(keys.ServerKeyPair{}, authSession)

	response := performJSONRequestWithToken(t, router, http.MethodPost, "/api/v1/users/me/signatures/verify", dto.VerifyUserSignatureRequest{
		Message:         "user signed message",
		SignatureBase64: base64.StdEncoding.EncodeToString([]byte("signature")),
	}, authSession.token)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}

	var body dto.VerifyClientSignatureResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.SignerType != "user" {
		t.Fatalf("expected signer_type user, got %q", body.SignerType)
	}
	if body.SignerUserID != authSession.user.ID {
		t.Fatalf("expected signer_user_id %q, got %q", authSession.user.ID, body.SignerUserID)
	}
	if body.Error != "current user does not have a registered public key" {
		t.Fatalf("unexpected error: %q", body.Error)
	}
}

func TestVerifyClientSignatureRouteRejectsInvalidBase64(t *testing.T) {
	gin.SetMode(gin.TestMode)
	requestBody := dto.VerifyClientSignatureRequest{
		Message:         "message",
		SignatureBase64: "not-base64",
		PublicKey:       "public key",
	}
	router := setupRouterWithSignatureHandler(keys.ServerKeyPair{})
	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/signatures/verify", requestBody)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}

	var body dto.VerifyClientSignatureResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Valid {
		t.Fatal("expected invalid base64 response to be invalid")
	}
	if body.Error != "signature_base64 must be valid base64" {
		t.Fatalf("unexpected error: %q", body.Error)
	}
}

func TestVerifyClientSignatureRouteRejectsEmptyMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	requestBody := dto.VerifyClientSignatureRequest{
		Message:         "   ",
		SignatureBase64: base64.StdEncoding.EncodeToString([]byte("signature")),
		PublicKey:       "public key",
	}
	router := setupRouterWithSignatureHandler(keys.ServerKeyPair{})
	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/signatures/verify", requestBody)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}

	var body dto.VerifyClientSignatureResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error != "message is required" {
		t.Fatalf("unexpected error: %q", body.Error)
	}
}

func TestVerifyClientSignatureRouteRejectsEmptyPublicKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	requestBody := dto.VerifyClientSignatureRequest{
		Message:         "message",
		SignatureBase64: base64.StdEncoding.EncodeToString([]byte("signature")),
		PublicKey:       "   ",
	}
	router := setupRouterWithSignatureHandler(keys.ServerKeyPair{})
	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/signatures/verify", requestBody)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}

	var body dto.VerifyClientSignatureResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error != "public_key is required" {
		t.Fatalf("unexpected error: %q", body.Error)
	}
}

func TestIssueServerMessageRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, publicKey := generateECDSAKeyPairPEM(t)
	authSession := newTestAuthSession(t)
	router, messageRepository := setupProtectedRouterWithSignatureHandlerAndRepository(keys.ServerKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, authSession)

	requestBody := dto.IssueServerMessageRequest{Message: "server generated proof"}
	response := performJSONRequestWithToken(t, router, http.MethodPost, "/api/v1/server/messages", requestBody, authSession.token)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	body := decodeIssueServerMessageResponse(t, response)
	if body.MessageID != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("expected message_id from generator, got %q", body.MessageID)
	}
	if _, err := time.Parse(time.RFC3339Nano, body.CreatedAt); err != nil {
		t.Fatalf("parse created_at: %v", err)
	}
	if body.Message != requestBody.Message {
		t.Fatalf("expected message %q, got %q", requestBody.Message, body.Message)
	}
	if body.CreatedByUserID != authSession.user.ID {
		t.Fatalf("expected created_by_user_id %q, got %q", authSession.user.ID, body.CreatedByUserID)
	}
	if body.SignerType != "server" {
		t.Fatalf("expected signer_type server, got %q", body.SignerType)
	}
	if body.Algorithm != crypto.ECDSASHA256Algorithm {
		t.Fatalf("expected algorithm %q, got %q", crypto.ECDSASHA256Algorithm, body.Algorithm)
	}

	assertServerMessageSignature(t, body, publicKey)
	if len(messageRepository.messages) != 1 {
		t.Fatalf("expected 1 saved message, got %d", len(messageRepository.messages))
	}
	if messageRepository.messages[0].ID != body.MessageID {
		t.Fatalf("expected saved message id %q, got %q", body.MessageID, messageRepository.messages[0].ID)
	}
	if messageRepository.messages[0].CreatedByUserID != authSession.user.ID {
		t.Fatalf("expected saved created_by_user_id %q, got %q", authSession.user.ID, messageRepository.messages[0].CreatedByUserID)
	}
	if len(messageRepository.messages[0].Signature) == 0 {
		t.Fatal("expected saved message signature")
	}
}

func TestGetServerMessageRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, publicKey := generateECDSAKeyPairPEM(t)
	authSession := newTestAuthSession(t)
	router, _ := setupProtectedRouterWithSignatureHandlerAndRepository(keys.ServerKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, authSession)

	createResponse := performJSONRequestWithToken(t, router, http.MethodPost, "/api/v1/server/messages", dto.IssueServerMessageRequest{
		Message: "traceable server message",
	}, authSession.token)
	if createResponse.Code != http.StatusOK {
		t.Fatalf("expected create status %d, got %d", http.StatusOK, createResponse.Code)
	}
	created := decodeIssueServerMessageResponse(t, createResponse)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/server/messages/"+created.MessageID, nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	body := decodeIssueServerMessageResponse(t, response)
	if body.MessageID != created.MessageID {
		t.Fatalf("expected message_id %q, got %q", created.MessageID, body.MessageID)
	}
	if body.Message != created.Message {
		t.Fatalf("expected message %q, got %q", created.Message, body.Message)
	}
	if body.HashBase64 != created.HashBase64 {
		t.Fatalf("expected hash %q, got %q", created.HashBase64, body.HashBase64)
	}
	if body.SignatureBase64 != created.SignatureBase64 {
		t.Fatalf("expected signature %q, got %q", created.SignatureBase64, body.SignatureBase64)
	}
	if body.CreatedByUserID != authSession.user.ID {
		t.Fatalf("expected created_by_user_id %q, got %q", authSession.user.ID, body.CreatedByUserID)
	}
	if body.SignerType != "server" {
		t.Fatalf("expected signer_type server, got %q", body.SignerType)
	}

	assertServerMessageSignature(t, body, publicKey)
}

func TestGetServerMessageRouteReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupRouterWithSignatureHandler(keys.ServerKeyPair{})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/server/messages/missing", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestIssueServerMessageRouteGeneratesMessageWhenRequestIsEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, publicKey := generateECDSAKeyPairPEM(t)
	authSession := newTestAuthSession(t)
	router := setupProtectedRouterWithSignatureHandler(keys.ServerKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, authSession)

	response := performJSONRequestWithToken(t, router, http.MethodPost, "/api/v1/server/messages", dto.IssueServerMessageRequest{}, authSession.token)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	body := decodeIssueServerMessageResponse(t, response)
	if body.Message == "" {
		t.Fatal("expected generated message")
	}

	assertServerMessageSignature(t, body, publicKey)
}

func TestIssueServerMessageRouteGeneratesMessageWhenBodyIsEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, publicKey := generateECDSAKeyPairPEM(t)
	authSession := newTestAuthSession(t)
	router := setupProtectedRouterWithSignatureHandler(keys.ServerKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, authSession)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/server/messages", nil)
	request.Header.Set("Authorization", "Bearer "+authSession.token)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	body := decodeIssueServerMessageResponse(t, response)
	if body.Message == "" {
		t.Fatal("expected generated message")
	}

	assertServerMessageSignature(t, body, publicKey)
}

func TestIssueServerMessageRouteReturnsErrorWhenPrivateKeyIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authSession := newTestAuthSession(t)
	router := setupProtectedRouterWithSignatureHandler(keys.ServerKeyPair{}, authSession)

	response := performJSONRequestWithToken(t, router, http.MethodPost, "/api/v1/server/messages", dto.IssueServerMessageRequest{
		Message: "server generated proof",
	}, authSession.token)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["error"] != "server private key is not loaded" {
		t.Fatalf("unexpected error: %q", body["error"])
	}
}

func setupRouterWithSignatureHandler(serverKeys keys.ServerKeyPair) *gin.Engine {
	router, _ := setupRouterWithSignatureHandlerAndRepository(serverKeys)
	return router
}

func setupRouterWithSignatureHandlerAndRepository(serverKeys keys.ServerKeyPair) (*gin.Engine, *fakeMessageRepository) {
	return setupProtectedRouterWithSignatureHandlerAndRepository(serverKeys, nil)
}

func setupProtectedRouterWithSignatureHandler(serverKeys keys.ServerKeyPair, authSession *testAuthSession) *gin.Engine {
	router, _ := setupProtectedRouterWithSignatureHandlerAndRepository(serverKeys, authSession)
	return router
}

func setupProtectedRouterWithSignatureHandlerAndRepository(serverKeys keys.ServerKeyPair, authSession *testAuthSession) (*gin.Engine, *fakeMessageRepository) {
	signatureProvider := crypto.NewECDSASHA256Provider()
	messageRepository := &fakeMessageRepository{}

	router := SetupRouter(&container.AppContainer{
		AuthMiddleware: newAuthMiddlewareForSession(authSession),
		SignatureHandler: handler.NewSignatureHandler(
			serverKeys,
			usecase.NewVerifyClientSignatureUseCase(signatureProvider),
			usecase.NewIssueServerSignedMessageUseCase(
				serverKeys.PrivateKey,
				signatureProvider,
				messageRepository,
				fakeIDGenerator{id: "00000000-0000-4000-8000-000000000001"},
				"server",
			),
			usecase.NewGetServerSignedMessageUseCase(messageRepository),
		),
	})

	return router, messageRepository
}

func setupRouterWithDocumentHandler(privateKey []byte) (*gin.Engine, *fakeDocumentRepository, *fakeDocumentStorage) {
	return setupProtectedRouterWithDocumentHandler(privateKey, nil)
}

func setupProtectedRouterWithDocumentHandler(privateKey []byte, authSession *testAuthSession) (*gin.Engine, *fakeDocumentRepository, *fakeDocumentStorage) {
	documentRepository := &fakeDocumentRepository{}
	documentStorage := &fakeDocumentStorage{}
	signatureProvider := crypto.NewECDSASHA256Provider()

	router := SetupRouter(&container.AppContainer{
		AuthMiddleware: newAuthMiddlewareForSession(authSession),
		DocumentHandler: handler.NewDocumentHandler(
			usecase.NewUploadDocumentUseCase(
				documentRepository,
				documentStorage,
				fakeIDGenerator{id: "00000000-0000-4000-8000-000000000001"},
				docx.NewProcessor(),
				signatureProvider,
				privateKey,
			),
			nil,
			nil,
			nil,
		),
	})

	return router, documentRepository, documentStorage
}

func setupRouterWithVerifyDecryptPackageHandler(publicKey []byte) *gin.Engine {
	signatureProvider := crypto.NewECDSASHA256Provider()

	return SetupRouter(&container.AppContainer{
		DocumentHandler: handler.NewDocumentHandler(
			nil,
			nil,
			nil,
			usecase.NewVerifyDecryptPackageUseCase(
				encryption.NewAESGCMEncryptor(),
				signatureProvider,
				publicKey,
			),
		),
	})
}

func setupRouterWithUserHandler(userRepository *fakeUserRepository) *gin.Engine {
	return setupProtectedRouterWithUserHandler(userRepository, nil)
}

func setupProtectedRouterWithUserHandler(userRepository *fakeUserRepository, authSession *testAuthSession) *gin.Engine {
	return SetupRouter(&container.AppContainer{
		AuthMiddleware: newAuthMiddlewareForSession(authSession),
		UserHandler: handler.NewUserHandler(
			usecase.NewRegisterUserUseCase(userRepository, fakeIDGenerator{id: "user-id"}),
			usecase.NewGetUserUseCase(userRepository),
			usecase.NewUpdateCurrentUserPublicKeyUseCase(userRepository),
		),
	})
}

func setupRouterWithAuth(userRepository *fakeUserRepository) (*gin.Engine, *infraauth.JWTManager) {
	jwtManager := infraauth.NewJWTManager("test-jwt-secret", time.Hour)
	currentUserUseCase := usecase.NewCurrentUserUseCase(userRepository)

	return SetupRouter(&container.AppContainer{
		AuthHandler: handler.NewAuthHandler(
			usecase.NewLoginUseCase(userRepository, jwtManager),
			currentUserUseCase,
		),
		AuthMiddleware: handler.NewAuthMiddleware(jwtManager, currentUserUseCase),
	}), jwtManager
}

type testAuthSession struct {
	user           model.User
	userRepository *fakeUserRepository
	jwtManager     *infraauth.JWTManager
	token          string
}

func newTestAuthSession(t *testing.T) *testAuthSession {
	t.Helper()

	user := model.User{
		ID:           "user-id",
		Email:        "user@example.com",
		Name:         "Lab User",
		PasswordHash: "hash",
		CreatedAt:    time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
	}
	userRepository := &fakeUserRepository{users: []model.User{user}}
	jwtManager := infraauth.NewJWTManager("test-jwt-secret", time.Hour)
	token, _, err := jwtManager.Generate(user.ID, user.Email)
	if err != nil {
		t.Fatalf("generate test access token: %v", err)
	}

	return &testAuthSession{
		user:           user,
		userRepository: userRepository,
		jwtManager:     jwtManager,
		token:          token,
	}
}

func newTestAuthSessionWithPublicKey(t *testing.T, publicKeyPEM string) *testAuthSession {
	t.Helper()

	authSession := newTestAuthSession(t)
	authSession.user.PublicKeyPEM = strings.TrimSpace(publicKeyPEM)
	authSession.userRepository.users[0].PublicKeyPEM = authSession.user.PublicKeyPEM
	authSession.userRepository.keyHistory = append(authSession.userRepository.keyHistory, model.UserKeyHistory{
		UserID:       authSession.user.ID,
		PublicKeyPEM: authSession.user.PublicKeyPEM,
		CreatedAt:    authSession.user.CreatedAt,
	})
	return authSession
}

func newAuthMiddlewareForSession(authSession *testAuthSession) *handler.AuthMiddleware {
	if authSession == nil {
		return nil
	}

	return handler.NewAuthMiddleware(authSession.jwtManager, usecase.NewCurrentUserUseCase(authSession.userRepository))
}

func performJSONRequest(t *testing.T, router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	return response
}

func performJSONRequestWithToken(t *testing.T, router *gin.Engine, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	return response
}

func performMultipartDocumentUpload(t *testing.T, router *gin.Engine, fileName string) *httptest.ResponseRecorder {
	t.Helper()

	return performMultipartDocumentUploadWithOptions(
		t,
		router,
		fileName,
		minimalDocx(t),
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	)
}

func performMultipartDocumentUploadWithToken(t *testing.T, router *gin.Engine, fileName, token string) *httptest.ResponseRecorder {
	t.Helper()

	return performMultipartDocumentUploadWithOptionsAndToken(
		t,
		router,
		fileName,
		minimalDocx(t),
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		token,
	)
}

func performMultipartDocumentUploadWithOptions(t *testing.T, router *gin.Engine, fileName string, content []byte, contentType string) *httptest.ResponseRecorder {
	t.Helper()

	return performMultipartDocumentUploadWithOptionsAndToken(t, router, fileName, content, contentType, "")
}

func performMultipartDocumentUploadWithOptionsAndToken(t *testing.T, router *gin.Engine, fileName string, content []byte, contentType string, token string) *httptest.ResponseRecorder {
	t.Helper()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	if err := writer.WriteField("owner_email", "owner@example.com"); err != nil {
		t.Fatalf("write owner_email field: %v", err)
	}
	if err := writer.WriteField("recipient_email", "recipient@example.com"); err != nil {
		t.Fatalf("write recipient_email field: %v", err)
	}

	fileHeader := make(textproto.MIMEHeader)
	fileHeader.Set("Content-Disposition", `form-data; name="file"; filename="`+fileName+`"`)
	if contentType != "" {
		fileHeader.Set("Content-Type", contentType)
	}

	fileWriter, err := writer.CreatePart(fileHeader)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.Copy(fileWriter, bytes.NewReader(content)); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents", &requestBody)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	return response
}

func decodeIssueServerMessageResponse(t *testing.T, response *httptest.ResponseRecorder) dto.IssueServerMessageResponse {
	t.Helper()

	var body dto.IssueServerMessageResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	return body
}

func encryptedPackageContent(t *testing.T, privateKey []byte, documentContent []byte) []byte {
	t.Helper()

	signatureProvider := crypto.NewECDSASHA256Provider()
	signature, err := signatureProvider.Sign(documentContent, privateKey)
	if err != nil {
		t.Fatalf("sign document content: %v", err)
	}

	pkg, err := encryption.NewAESGCMEncryptor().EncryptDocument(model.Document{
		ID:               "document-id",
		OriginalFileName: "contract.docx",
		MimeType:         "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Hash:             signatureProvider.Hash(documentContent),
		Signature:        signature,
	}, documentContent)
	if err != nil {
		t.Fatalf("encrypt document package: %v", err)
	}

	encodedPackage, err := encryption.EncodePackage(pkg)
	if err != nil {
		t.Fatalf("encode package: %v", err)
	}

	return encodedPackage
}

func tamperedPackageSignatureContent(t *testing.T, packageContent []byte) []byte {
	t.Helper()

	pkg, err := encryption.DecodePackage(packageContent)
	if err != nil {
		t.Fatalf("decode package: %v", err)
	}

	signature, err := base64.StdEncoding.DecodeString(pkg.SignatureBase64)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	signature[len(signature)-1] ^= 0xff
	pkg.SignatureBase64 = base64.StdEncoding.EncodeToString(signature)

	encodedPackage, err := encryption.EncodePackage(pkg)
	if err != nil {
		t.Fatalf("encode tampered package: %v", err)
	}

	return encodedPackage
}

func tamperedPackageCiphertextContent(t *testing.T, packageContent []byte) []byte {
	t.Helper()

	pkg, err := encryption.DecodePackage(packageContent)
	if err != nil {
		t.Fatalf("decode package: %v", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(pkg.CiphertextBase64)
	if err != nil {
		t.Fatalf("decode ciphertext: %v", err)
	}
	ciphertext[len(ciphertext)-1] ^= 0xff
	pkg.CiphertextBase64 = base64.StdEncoding.EncodeToString(ciphertext)

	encodedPackage, err := encryption.EncodePackage(pkg)
	if err != nil {
		t.Fatalf("encode tampered package: %v", err)
	}

	return encodedPackage
}

func assertServerMessageSignature(t *testing.T, body dto.IssueServerMessageResponse, publicKey []byte) {
	t.Helper()

	provider := crypto.NewECDSASHA256Provider()

	messageHash, err := base64.StdEncoding.DecodeString(body.HashBase64)
	if err != nil {
		t.Fatalf("decode hash_base64: %v", err)
	}
	expectedHash := provider.Hash([]byte(body.Message))
	if string(messageHash) != string(expectedHash) {
		t.Fatalf("expected hash %x, got %x", expectedHash, messageHash)
	}

	signature, err := base64.StdEncoding.DecodeString(body.SignatureBase64)
	if err != nil {
		t.Fatalf("decode signature_base64: %v", err)
	}
	if err := provider.Verify([]byte(body.Message), signature, publicKey); err != nil {
		t.Fatalf("verify server message signature: %v", err)
	}
}

func assertDocumentSignature(t *testing.T, content []byte, document model.Document, publicKey []byte) {
	t.Helper()

	provider := crypto.NewECDSASHA256Provider()
	expectedHash := provider.Hash(content)
	if string(document.Hash) != string(expectedHash) {
		t.Fatalf("expected document hash %x, got %x", expectedHash, document.Hash)
	}
	if err := provider.Verify(content, document.Signature, publicKey); err != nil {
		t.Fatalf("verify document signature: %v", err)
	}
}

func generateECDSAKeyPairPEM(t *testing.T) ([]byte, []byte) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}

	privateKeyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}

	publicKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}

	return pemBlock("EC PRIVATE KEY", privateKeyDER), pemBlock("PUBLIC KEY", publicKeyDER)
}

func pemBlock(blockType string, bytes []byte) []byte {
	return []byte("-----BEGIN " + blockType + "-----\n" +
		base64.StdEncoding.EncodeToString(bytes) +
		"\n-----END " + blockType + "-----\n")
}

func minimalDocx(t *testing.T) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)

	documentXML, err := writer.Create("word/document.xml")
	if err != nil {
		t.Fatalf("create document.xml: %v", err)
	}
	if _, err := documentXML.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Hello</w:t></w:r></w:p></w:body></w:document>`)); err != nil {
		t.Fatalf("write document.xml: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close docx zip: %v", err)
	}

	return buffer.Bytes()
}

func readDocxDocumentXML(t *testing.T, content []byte) string {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("open stored docx: %v", err)
	}

	for _, file := range reader.File {
		if file.Name != "word/document.xml" {
			continue
		}

		source, err := file.Open()
		if err != nil {
			t.Fatalf("open document.xml: %v", err)
		}
		defer source.Close()

		documentXML, err := io.ReadAll(source)
		if err != nil {
			t.Fatalf("read document.xml: %v", err)
		}

		return string(documentXML)
	}

	t.Fatal("word/document.xml not found")
	return ""
}

type fakeMessageRepository struct {
	messages []model.Message
}

func (r *fakeMessageRepository) Create(_ context.Context, message *model.Message) error {
	r.messages = append(r.messages, *message)
	return nil
}

func (r *fakeMessageRepository) FindByID(_ context.Context, id string) (*model.Message, error) {
	for i := range r.messages {
		if r.messages[i].ID == id {
			return &r.messages[i], nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}

type fakeDocumentRepository struct {
	documents []model.Document
}

func (r *fakeDocumentRepository) Create(_ context.Context, document *model.Document) error {
	r.documents = append(r.documents, *document)
	return nil
}

func (r *fakeDocumentRepository) FindByID(_ context.Context, id string) (*model.Document, error) {
	for i := range r.documents {
		if r.documents[i].ID == id {
			return &r.documents[i], nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}

func (r *fakeDocumentRepository) Update(_ context.Context, document *model.Document) error {
	for i := range r.documents {
		if r.documents[i].ID == document.ID {
			r.documents[i] = *document
			return nil
		}
	}

	r.documents = append(r.documents, *document)
	return nil
}

type fakeDocumentStorage struct {
	content                 []byte
	encryptedPackageContent []byte
}

func (s *fakeDocumentStorage) Save(_ context.Context, _ string, originalFileName string, content io.Reader) (string, error) {
	storedContent, err := io.ReadAll(content)
	if err != nil {
		return "", err
	}

	s.content = storedContent
	return "stored/" + originalFileName, nil
}

func (s *fakeDocumentStorage) Read(_ context.Context, path string) ([]byte, error) {
	if strings.HasSuffix(path, "_encrypted_package.json") {
		return s.encryptedPackageContent, nil
	}

	return s.content, nil
}

func (s *fakeDocumentStorage) SaveEncryptedPackage(_ context.Context, documentID string, content []byte) (string, error) {
	s.encryptedPackageContent = content
	return "stored/" + documentID + "_encrypted_package.json", nil
}

type fakeMailer struct {
	to          []string
	subject     string
	body        string
	attachments []usecase.EmailAttachment
	err         error
}

func (m *fakeMailer) SendEmail(_ context.Context, to []string, subject, body string, attachments []usecase.EmailAttachment) error {
	m.to = to
	m.subject = subject
	m.body = body
	m.attachments = attachments
	return m.err
}

type fakeUserRepository struct {
	users      []model.User
	keyHistory []model.UserKeyHistory
}

func (r *fakeUserRepository) Create(_ context.Context, user *model.User) error {
	r.users = append(r.users, *user)
	return nil
}

func (r *fakeUserRepository) FindByID(_ context.Context, id string) (*model.User, error) {
	for i := range r.users {
		if r.users[i].ID == id {
			return &r.users[i], nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}

func (r *fakeUserRepository) FindByEmail(_ context.Context, email string) (*model.User, error) {
	for i := range r.users {
		if r.users[i].Email == email {
			return &r.users[i], nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}

func (r *fakeUserRepository) Update(_ context.Context, user *model.User) error {
	for i := range r.users {
		if r.users[i].ID == user.ID {
			r.users[i] = *user
			return nil
		}
	}

	r.users = append(r.users, *user)
	return nil
}

func (r *fakeUserRepository) CreateKeyHistory(_ context.Context, entry *model.UserKeyHistory) error {
	r.keyHistory = append(r.keyHistory, *entry)
	return nil
}

type fakeIDGenerator struct {
	id string
}

func (g fakeIDGenerator) Generate() (string, error) {
	return g.id, nil
}
