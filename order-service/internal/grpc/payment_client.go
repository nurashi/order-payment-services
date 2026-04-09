package grpc

import (
	"context"
	"fmt"
	"time"

	paymentv1 "github.com/nurashi/order-service/gen/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type grpcPaymentClient struct {
	client paymentv1.PaymentServiceClient
}

func NewPaymentClient(address string) (*grpcPaymentClient, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to payment service: %w", err)
	}
	return &grpcPaymentClient{client: paymentv1.NewPaymentServiceClient(conn)}, nil
}

func (c *grpcPaymentClient) ProcessPayment(orderID string, amount int64) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.ProcessPayment(ctx, &paymentv1.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		return "", fmt.Errorf("payment service unavailable: %w", err)
	}

	return resp.Status, nil
}
