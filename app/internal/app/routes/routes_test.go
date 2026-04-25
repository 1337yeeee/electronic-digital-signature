package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"electronic-digital-signature/internal/app/container"
	"electronic-digital-signature/internal/app/dto"
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
	router := SetupRouter(&container.AppContainer{
		ServerKeys: keys.ServerKeyPair{PublicKey: publicKey},
	})

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
	router := SetupRouter(&container.AppContainer{})

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
