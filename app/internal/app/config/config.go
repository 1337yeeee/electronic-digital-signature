package config

import (
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

	ServerKeys ServerKeysConfig
	Redis      RedisConfig
}

type ServerKeysConfig struct {
	PrivateKeyPath string
	PublicKeyPath  string
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
		APIPort: os.Getenv("API_PORT"),

		DBHost:     os.Getenv("DB_HOST"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		DBPort:     os.Getenv("DB_PORT"),
		SSLMode:    os.Getenv("SSL_MODE"),

		JWTSecret: os.Getenv("JWT_SECRET"),

		ServerKeys: ServerKeysConfig{
			PrivateKeyPath: os.Getenv("SERVER_PRIVATE_KEY_PATH"),
			PublicKeyPath:  os.Getenv("SERVER_PUBLIC_KEY_PATH"),
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

func (c Config) Get(key string) string {
	return os.Getenv(key)
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
