package config

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	PaymentService PaymentServiceConfig
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
	}

	return cfg, nil
}

// GRPCListenAddr returns host:port for the order gRPC server from ORDER_GRPC_HOST and ORDER_GRPC_PORT.
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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
