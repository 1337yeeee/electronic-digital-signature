package routes

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"electronic-digital-signature/internal/app/container"
	"electronic-digital-signature/internal/app/dto"
	"electronic-digital-signature/internal/app/handler"
	"electronic-digital-signature/internal/app/usecase"
	"electronic-digital-signature/internal/infra/crypto"
	"electronic-digital-signature/internal/infra/keys"

	"github.com/gin-gonic/gin"
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
	router := setupRouterWithSignatureHandler(keys.ServerKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	})

	requestBody := dto.IssueServerMessageRequest{Message: "server generated proof"}
	response := performJSONRequest(t, router, http.MethodPost, "/api/v1/server/messages", requestBody)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	body := decodeIssueServerMessageResponse(t, response)
	if body.ID == "" {
		t.Fatal("expected response id")
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
	signatureProvider := crypto.NewECDSASHA256Provider()

	return SetupRouter(&container.AppContainer{
		SignatureHandler: handler.NewSignatureHandler(
			serverKeys,
			usecase.NewVerifyClientSignatureUseCase(signatureProvider),
			usecase.NewIssueServerSignedMessageUseCase(serverKeys.PrivateKey, signatureProvider),
		),
	})
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
