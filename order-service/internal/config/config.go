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
	Server         ServerConfig
	Database       DatabaseConfig
	PaymentService PaymentServiceConfig
	Redis          RedisConfig
	RateLimiter    RateLimiterConfig
}

type ServerConfig struct {
	Port     string
	GRPCHost string
	GRPCPort string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type PaymentServiceConfig struct {
	GRPCAddress string
}

type RedisConfig struct {
	Host            string
	Port            string
	CacheTTLSeconds int
}

type RateLimiterConfig struct {
	MaxRequests   int
	WindowSeconds int
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:     getEnv("PORT", "8080"),
			GRPCHost: getEnv("ORDER_GRPC_HOST", "0.0.0.0"),
			GRPCPort: getEnv("ORDER_GRPC_PORT", "9090"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "order_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		PaymentService: PaymentServiceConfig{
			GRPCAddress: getEnv("PAYMENT_GRPC_ADDRESS", "localhost:9091"),
		},
		Redis: RedisConfig{
			Host:            getEnv("REDIS_HOST", "localhost"),
			Port:            getEnv("REDIS_PORT", "6379"),
			CacheTTLSeconds: getEnvInt("CACHE_TTL_SECONDS", 300),
		},
		RateLimiter: RateLimiterConfig{
			MaxRequests:   getEnvInt("RATE_LIMIT_MAX_REQUESTS", 10),
			WindowSeconds: getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60),
		},
	}

	return cfg, nil
}

func (c *Config) GRPCListenAddr() string {
	return net.JoinHostPort(c.Server.GRPCHost, c.Server.GRPCPort)
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
