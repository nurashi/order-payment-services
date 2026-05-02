package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nurashi/notification-service/internal/config"
	"github.com/nurashi/notification-service/internal/messaging/rabbitmq"
	"github.com/nurashi/notification-service/internal/migration"
	"github.com/nurashi/notification-service/internal/repository"
	"github.com/nurashi/notification-service/internal/service"

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

	log.Println("Connected to notification database successfully")

	if err := migration.Run(cfg.GetDSN()); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations applied successfully")

	consumer, err := rabbitmq.NewRabbitMQConsumer(
		cfg.RabbitMQ.Host,
		cfg.RabbitMQ.Port,
		cfg.RabbitMQ.User,
		cfg.RabbitMQ.Password,
		"payment.completed",
	)
	if err != nil {
		log.Fatalf("Failed to create RabbitMQ consumer: %v", err)
	}

	idempotencyRepo := repository.NewIdempotencyRepository(dbpool)
	notificationSvc := service.NewNotificationService(idempotencyRepo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := consumer.Start(ctx, notificationSvc); err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down notification service...")

	cancel()
	if err := consumer.Stop(); err != nil {
		log.Printf("Error stopping consumer: %v", err)
	}

	log.Println("Notification service stopped")
}
