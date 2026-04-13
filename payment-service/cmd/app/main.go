package main

import (
	"context"
	"log"
	"net"

	paymentpb "github.com/nurashi/ap2-generated/payment/v1"
	"github.com/nurashi/payment-service/internal/api"
	"github.com/nurashi/payment-service/internal/config"
	grpcserver "github.com/nurashi/payment-service/internal/grpc"
	"github.com/nurashi/payment-service/internal/migration"
	"github.com/nurashi/payment-service/internal/repository"
	"github.com/nurashi/payment-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
	paymentSvc := service.NewPaymentService(paymentRepo)

	go func() {
		addr := cfg.GRPCListenAddr()
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("Failed to listen on gRPC %s: %v", addr, err)
		}
		grpcSrv := grpc.NewServer(
			grpc.UnaryInterceptor(grpcserver.LoggingInterceptor),
		)
		paymentpb.RegisterPaymentServiceServer(grpcSrv, grpcserver.NewPaymentServer(paymentSvc))
		reflection.Register(grpcSrv)
		log.Printf("Payment gRPC server listening on %s", addr)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	paymentHandler := api.NewPaymentHandler(paymentSvc)
	router := gin.Default()
	paymentHandler.RegisterRoutes(router)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "payment-service"})
	})

	log.Printf("Payment HTTP server starting on port %s", cfg.Server.Port)
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
