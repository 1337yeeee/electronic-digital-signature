package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	//TODO Config
	APIPort string

	DBHost     string
	DBUser     string
	DBPassword string
	DBName     string
	DBPort     string
	SSLMode    string

	JWTSecret string

	ServerKeys      ServerKeysConfig
	DocumentStorage DocumentStorageConfig
	SMTP            SMTPConfig
	Redis           RedisConfig
}

type ServerKeysConfig struct {
	PrivateKeyPath string
	PublicKeyPath  string
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

type RedisConfig struct {
	Addr        string
	Password    string
	User        string
	DB          int
	MaxRetries  int
	DialTimeout time.Duration
	Timeout     time.Duration
}

func Load() (Config, error) {
	return Config{
		APIPort: getEnv("API_PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     getEnv("DB_NAME", "eds_lab"),
		DBPort:     getEnv("DB_PORT", "5432"),
		SSLMode:    getEnv("SSL_MODE", "disable"),

		JWTSecret: os.Getenv("JWT_SECRET"),

		ServerKeys: ServerKeysConfig{
			PrivateKeyPath: os.Getenv("SERVER_PRIVATE_KEY_PATH"),
			PublicKeyPath:  os.Getenv("SERVER_PUBLIC_KEY_PATH"),
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

		Redis: RedisConfig{
			Addr:     os.Getenv("REDIS_ADDR"),
			Password: os.Getenv("REDIS_PASSWORD"),
			User:     os.Getenv("REDIS_USER"),
			DB:       getEnvAsInt("REDIS_DB", 0),

			MaxRetries:  getEnvAsInt("REDIS_MAX_RETRIES", 3),
			DialTimeout: getEnvAsDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			Timeout:     getEnvAsDuration("REDIS_TIMEOUT", 3*time.Second),
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

func getEnvAsInt(key string, def int) int {
	if val, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
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
