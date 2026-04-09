package main

import (
	"context"
	"log"

	"github.com/nurashi/payment-service/internal/api"
	"github.com/nurashi/payment-service/internal/config"
	"github.com/nurashi/payment-service/internal/migration"
	"github.com/nurashi/payment-service/internal/repository"
	"github.com/nurashi/payment-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dbpool, err := pgxpool.New(context.Background(), cfg.GetDSN())
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}
	defer dbpool.Close()

	if err := dbpool.Ping(context.Background()); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	log.Println("Connected to payment database successfully")


	if err := migration.Run(cfg.GetDSN()); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations applied successfully")

	paymentRepo := repository.NewPaymentRepository(dbpool)
	paymentService := service.NewPaymentService(paymentRepo)
	paymentHandler := api.NewPaymentHandler(paymentService)

	router := gin.Default()

	paymentHandler.RegisterRoutes(router)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "payment-service"})
	})

	log.Printf("Payment Service starting on port %s", cfg.Server.Port)
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
