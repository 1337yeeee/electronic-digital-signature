package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

type Claims struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Expires int64  `json:"exp"`
}

type JWTManager struct {
	secret   []byte
	tokenTTL time.Duration
}

func NewJWTManager(secret string, tokenTTL time.Duration) *JWTManager {
	return &JWTManager{
		secret:   []byte(secret),
		tokenTTL: tokenTTL,
	}
}

func (m *JWTManager) Generate(subject, email string) (string, time.Time, error) {
	if len(m.secret) == 0 {
		return "", time.Time{}, fmt.Errorf("jwt secret is empty")
	}

	expiresAt := time.Now().UTC().Add(m.tokenTTL)
	headerJSON, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("marshal jwt header: %w", err)
	}

	payloadJSON, err := json.Marshal(Claims{
		Subject: subject,
		Email:   email,
		Expires: expiresAt.Unix(),
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("marshal jwt payload: %w", err)
	}

	encodedHeader := encodeSegment(headerJSON)
	encodedPayload := encodeSegment(payloadJSON)
	unsignedToken := encodedHeader + "." + encodedPayload
	signature := m.sign(unsignedToken)

	return unsignedToken + "." + encodeSegment(signature), expiresAt, nil
}

func (m *JWTManager) Verify(token string) (Claims, error) {
	if len(m.secret) == 0 {
		return Claims{}, fmt.Errorf("jwt secret is empty")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	unsignedToken := parts[0] + "." + parts[1]
	providedSignature, err := decodeSegment(parts[2])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	expectedSignature := m.sign(unsignedToken)
	if !hmac.Equal(providedSignature, expectedSignature) {
		return Claims{}, ErrInvalidToken
	}

	payloadJSON, err := decodeSegment(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return Claims{}, ErrInvalidToken
	}
	if time.Now().UTC().Unix() >= claims.Expires {
		return Claims{}, ErrExpiredToken
	}

	return claims, nil
}

func (m *JWTManager) sign(value string) []byte {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func encodeSegment(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}

func decodeSegment(value string) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}
