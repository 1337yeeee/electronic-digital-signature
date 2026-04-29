package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	APIPort string
	CORS    CORSConfig

	DBHost     string
	DBUser     string
	DBPassword string
	DBName     string
	DBPort     string
	SSLMode    string

	ServerKeys      ServerKeysConfig
	DocumentStorage DocumentStorageConfig
	SMTP            SMTPConfig
	Auth            AuthConfig
}

type ServerKeysConfig struct {
	PrivateKeyPath string
	PublicKeyPath  string
	PrivateKeyPEM  string
	PublicKeyPEM   string
}

type DocumentStorageConfig struct {
	Path string
}

type SMTPConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

type AuthConfig struct {
	JWTSecret string
	TokenTTL  time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
}

func Load() (Config, error) {
	return Config{
		APIPort: getEnv("API_PORT", "8080"),
		CORS: CORSConfig{
			AllowedOrigins: getEnvAsCSV(
				"CORS_ALLOWED_ORIGINS",
				[]string{
					"http://localhost:3000",
					"http://127.0.0.1:3000",
					"http://localhost:8080",
					"http://127.0.0.1:8080",
				},
			),
		},

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     getEnv("DB_NAME", "eds_lab"),
		DBPort:     getEnv("DB_PORT", "5432"),
		SSLMode:    getEnv("SSL_MODE", "disable"),

		ServerKeys: ServerKeysConfig{
			PrivateKeyPath: os.Getenv("SERVER_PRIVATE_KEY_PATH"),
			PublicKeyPath:  os.Getenv("SERVER_PUBLIC_KEY_PATH"),
			PrivateKeyPEM:  os.Getenv("SERVER_PRIVATE_KEY_PEM"),
			PublicKeyPEM:   os.Getenv("SERVER_PUBLIC_KEY_PEM"),
		},

		DocumentStorage: DocumentStorageConfig{
			Path: getEnv("DOCUMENT_STORAGE_PATH", "data/uploads"),
		},

		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", "localhost"),
			Port:     getEnv("SMTP_PORT", "1025"),
			User:     os.Getenv("SMTP_USER"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     getEnv("SMTP_FROM", "server@example.com"),
		},

		Auth: AuthConfig{
			JWTSecret: getEnv("AUTH_JWT_SECRET", "dev-jwt-secret"),
			TokenTTL:  getEnvAsDuration("AUTH_TOKEN_TTL", 24*time.Hour),
		},
	}, nil
}

func (c Config) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.DBHost,
		c.DBUser,
		c.DBPassword,
		c.DBName,
		c.DBPort,
		c.SSLMode,
	)
}

func (c Config) Get(key string) string {
	return os.Getenv(key)
}

func getEnv(key, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func getEnvAsDuration(key string, def time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok {
		if parsed, err := time.ParseDuration(val); err == nil {
			return parsed
		}
	}

	return def
}

func getEnvAsCSV(key string, def []string) []string {
	if val, ok := os.LookupEnv(key); ok {
		parts := strings.Split(val, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	return def
}
