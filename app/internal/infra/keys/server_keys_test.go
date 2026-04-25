package keys

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadServerKeyPair(t *testing.T) {
	dir := t.TempDir()
	privateKeyPath := filepath.Join(dir, "server_private.pem")
	publicKeyPath := filepath.Join(dir, "server_public.pem")

	writeTestFile(t, privateKeyPath, []byte("private key"))
	writeTestFile(t, publicKeyPath, []byte("public key"))

	keyPair, err := LoadServerKeyPair(privateKeyPath, publicKeyPath)
	if err != nil {
		t.Fatalf("load server key pair: %v", err)
	}

	if string(keyPair.PrivateKey) != "private key" {
		t.Fatalf("unexpected private key: %q", keyPair.PrivateKey)
	}
	if string(keyPair.PublicKey) != "public key" {
		t.Fatalf("unexpected public key: %q", keyPair.PublicKey)
	}
}

func TestLoadServerKeyPairRejectsEmptyPath(t *testing.T) {
	_, err := LoadServerKeyPair("", "server_public.pem")
	if err == nil {
		t.Fatal("expected empty private key path to fail")
	}
	if !strings.Contains(err.Error(), "server private key path is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadServerKeyPairRejectsMissingFile(t *testing.T) {
	_, err := LoadServerKeyPair(filepath.Join(t.TempDir(), "missing.pem"), "server_public.pem")
	if err == nil {
		t.Fatal("expected missing private key file to fail")
	}
	if !strings.Contains(err.Error(), "server private key file does not exist") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadServerKeyPairRejectsEmptyFile(t *testing.T) {
	dir := t.TempDir()
	privateKeyPath := filepath.Join(dir, "server_private.pem")
	publicKeyPath := filepath.Join(dir, "server_public.pem")

	writeTestFile(t, privateKeyPath, nil)
	writeTestFile(t, publicKeyPath, []byte("public key"))

	_, err := LoadServerKeyPair(privateKeyPath, publicKeyPath)
	if err == nil {
		t.Fatal("expected empty private key file to fail")
	}
	if !strings.Contains(err.Error(), "server private key file is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeTestFile(t *testing.T, path string, content []byte) {
	t.Helper()

	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write test file %q: %v", path, err)
	}
}
