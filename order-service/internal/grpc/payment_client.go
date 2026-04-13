package grpc

import (
	"context"
	"fmt"
	"time"

	paymentpb "github.com/nurashi/ap2-generated/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type grpcPaymentClient struct {
	client paymentpb.PaymentServiceClient
}

func NewPaymentClient(address string) (*grpcPaymentClient, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to payment service: %w", err)
	}
	return &grpcPaymentClient{client: paymentpb.NewPaymentServiceClient(conn)}, nil
}

func (c *grpcPaymentClient) ProcessPayment(orderID string, amount int64) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.ProcessPayment(ctx, &paymentpb.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return "", fmt.Errorf("payment service: %s (%s)", st.Message(), st.Code().String())
		}
		return "", fmt.Errorf("payment service unavailable: %w", err)
	}

	return resp.Status, nil
}
