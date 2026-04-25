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
	"strings"
	"testing"
	"time"

	"electronic-digital-signature/internal/app/container"
	"electronic-digital-signature/internal/app/dto"
	"electronic-digital-signature/internal/app/handler"
	"electronic-digital-signature/internal/app/usecase"
	"electronic-digital-signature/internal/domain/model"
	"electronic-digital-signature/internal/infra/crypto"
	"electronic-digital-signature/internal/infra/docx"
	"electronic-digital-signature/internal/infra/keys"

	"github.com/gin-gonic/gin"
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

	expectedBody := `{"status":"ok"}`
	if response.Body.String() != expectedBody {
		t.Fatalf("expected body %s, got %s", expectedBody, response.Body.String())
	}
}

func TestUploadDocumentRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, publicKey := generateECDSAKeyPairPEM(t)
	router, documentRepository, documentStorage := setupRouterWithDocumentHandler(privateKey)

	response := performMultipartDocumentUpload(t, router, "contract.docx")

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.Code)
	}

	var body dto.UploadDocumentResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.DocumentID != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("expected document_id from generator, got %q", body.DocumentID)
	}
	if body.OwnerEmail != "owner@example.com" {
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
	router, _, _ := setupRouterWithDocumentHandler(privateKey)

	response := performMultipartDocumentUpload(t, router, "contract.txt")

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["error"] != "document file must have .docx extension" {
		t.Fatalf("unexpected error: %q", body["error"])
	}
}

func TestSendDocumentRouteSendsEncryptedPackageAndStoresStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	documentRepository := &fakeDocumentRepository{
		documents: []model.Document{
			{
				ID:               "document-id",
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
		DocumentHandler: handler.NewDocumentHandler(
			nil,
			usecase.NewSendDocumentUseCase(documentRepository, documentStorage, nil, mailer),
		),
	})

	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/documents/document-id/send", dto.SendDocumentRequest{
		Email: "recipient@example.com",
	})

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}

	var body dto.SendDocumentResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.DocumentID != "document-id" {
		t.Fatalf("expected document_id, got %q", body.DocumentID)
	}
	if body.RecipientEmail != "recipient@example.com" {
		t.Fatalf("expected recipient email, got %q", body.RecipientEmail)
	}
	if body.SendStatus != usecase.DocumentSendStatusSent {
		t.Fatalf("expected send status sent, got %q", body.SendStatus)
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
	if savedDocument.LastSentToEmail != "recipient@example.com" {
		t.Fatalf("expected saved sent email, got %q", savedDocument.LastSentToEmail)
	}
	if savedDocument.SentAt == nil {
		t.Fatal("expected saved sent_at")
	}
}

func TestSendDocumentRouteReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := SetupRouter(&container.AppContainer{
		DocumentHandler: handler.NewDocumentHandler(
			nil,
			usecase.NewSendDocumentUseCase(&fakeDocumentRepository{}, &fakeDocumentStorage{}, nil, &fakeMailer{}),
		),
	})

	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/documents/missing/send", dto.SendDocumentRequest{
		Email: "recipient@example.com",
	})

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
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

func TestIssueServerMessageRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, publicKey := generateECDSAKeyPairPEM(t)
	router, messageRepository := setupRouterWithSignatureHandlerAndRepository(keys.ServerKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	})

	requestBody := dto.IssueServerMessageRequest{Message: "server generated proof"}
	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/server/messages", requestBody)

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
	if len(messageRepository.messages[0].Signature) == 0 {
		t.Fatal("expected saved message signature")
	}
}

func TestGetServerMessageRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, publicKey := generateECDSAKeyPairPEM(t)
	router, _ := setupRouterWithSignatureHandlerAndRepository(keys.ServerKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	})

	createResponse := performJSONRequest(t, router, http.MethodPost, "/api/v1/server/messages", dto.IssueServerMessageRequest{
		Message: "traceable server message",
	})
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
	router := setupRouterWithSignatureHandler(keys.ServerKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	})

	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/server/messages", dto.IssueServerMessageRequest{})

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
	router := setupRouterWithSignatureHandler(keys.ServerKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/server/messages", nil)
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
	router := setupRouterWithSignatureHandler(keys.ServerKeyPair{})

	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/server/messages", dto.IssueServerMessageRequest{
		Message: "server generated proof",
	})

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
	signatureProvider := crypto.NewECDSASHA256Provider()
	messageRepository := &fakeMessageRepository{}

	router := SetupRouter(&container.AppContainer{
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
	documentRepository := &fakeDocumentRepository{}
	documentStorage := &fakeDocumentStorage{}
	signatureProvider := crypto.NewECDSASHA256Provider()

	router := SetupRouter(&container.AppContainer{
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
		),
	})

	return router, documentRepository, documentStorage
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

func performMultipartDocumentUpload(t *testing.T, router *gin.Engine, fileName string) *httptest.ResponseRecorder {
	t.Helper()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	if err := writer.WriteField("owner_email", "owner@example.com"); err != nil {
		t.Fatalf("write owner_email field: %v", err)
	}
	if err := writer.WriteField("recipient_email", "recipient@example.com"); err != nil {
		t.Fatalf("write recipient_email field: %v", err)
	}

	fileWriter, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.Copy(fileWriter, bytes.NewReader(minimalDocx(t))); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/documents", &requestBody)
	request.Header.Set("Content-Type", writer.FormDataContentType())
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

type fakeIDGenerator struct {
	id string
}

func (g fakeIDGenerator) Generate() (string, error) {
	return g.id, nil
}
