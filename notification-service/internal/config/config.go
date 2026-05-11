package config

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Database DatabaseConfig
	RabbitMQ RabbitMQConfig
	Redis    RedisConfig
	Provider ProviderConfig
	Retry    RetryConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RabbitMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

type RedisConfig struct {
	Host               string
	Port               string
	IdempotencyTTLSecs int
}

type ProviderConfig struct {
	Mode     string
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	FromAddr string
}

type RetryConfig struct {
	MaxAttempts           int
	InitialBackoffSeconds int
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "notification_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		RabbitMQ: RabbitMQConfig{
			Host:     getEnv("RABBITMQ_HOST", "localhost"),
			Port:     getEnv("RABBITMQ_PORT", "5672"),
			User:     getEnv("RABBITMQ_USER", "guest"),
			Password: getEnv("RABBITMQ_PASSWORD", "guest"),
		},
		Redis: RedisConfig{
			Host:               getEnv("REDIS_HOST", "localhost"),
			Port:               getEnv("REDIS_PORT", "6379"),
			IdempotencyTTLSecs: getEnvInt("IDEMPOTENCY_TTL_SECONDS", 86400),
		},
		Provider: ProviderConfig{
			Mode:     getEnv("PROVIDER_MODE", "SIMULATED"),
			SMTPHost: getEnv("SMTP_HOST", ""),
			SMTPPort: getEnv("SMTP_PORT", "587"),
			SMTPUser: getEnv("SMTP_USER", ""),
			SMTPPass: getEnv("SMTP_PASS", ""),
			FromAddr: getEnv("SMTP_FROM", ""),
		},
		Retry: RetryConfig{
			MaxAttempts:           getEnvInt("MAX_RETRY_ATTEMPTS", 3),
			InitialBackoffSeconds: getEnvInt("INITIAL_BACKOFF_SECONDS", 2),
		},
	}

	return cfg, nil
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

func (c *Config) RedisAddr() string {
	return net.JoinHostPort(c.Redis.Host, c.Redis.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

