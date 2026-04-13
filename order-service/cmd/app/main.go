package main

import (
	"context"
	"log"
	"net"

	orderpb "github.com/nurashi/ap2-generated/order/v1"
	"github.com/nurashi/order-service/internal/api"
	"github.com/nurashi/order-service/internal/config"
	grpcclient "github.com/nurashi/order-service/internal/grpc"
	"github.com/nurashi/order-service/internal/migration"
	"github.com/nurashi/order-service/internal/repository"
	"github.com/nurashi/order-service/internal/service"

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

	log.Println("Connected to order database successfully")

	if err := migration.Run(cfg.GetDSN()); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations applied successfully")

	paymentClient, err := grpcclient.NewPaymentClient(cfg.PaymentService.GRPCAddress)
	if err != nil {
		log.Fatalf("Failed to create payment gRPC client: %v", err)
	}

	orderRepo := repository.NewOrderRepository(dbpool)
	orderSubscriber := repository.NewOrderSubscriber(cfg.GetDSN(), orderRepo)
	orderSvc := service.NewOrderService(orderRepo, paymentClient)

	go func() {
		addr := cfg.GRPCListenAddr()
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("Failed to listen on gRPC %s: %v", addr, err)
		}
		grpcSrv := grpc.NewServer()
		orderpb.RegisterOrderServiceServer(grpcSrv, grpcclient.NewOrderServer(orderSubscriber))
		reflection.Register(grpcSrv)
		log.Printf("Order gRPC server listening on %s", addr)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("Order gRPC server failed: %v", err)
		}
	}()

	orderHandler := api.NewOrderHandler(orderSvc)
	router := gin.Default()
	orderHandler.RegisterRoutes(router)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "order-service"})
	})

	log.Printf("Order Service starting on port %s", cfg.Server.Port)
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
