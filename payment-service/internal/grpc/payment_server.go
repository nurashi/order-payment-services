package grpc

import (
	"context"
	"time"

	paymentv1 "github.com/nurashi/payment-service/gen/payment/v1"
	"github.com/nurashi/payment-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PaymentServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	svc service.PaymentService
}

func NewPaymentServer(svc service.PaymentService) *PaymentServer {
	return &PaymentServer{svc: svc}
}

func (s *PaymentServer) ProcessPayment(ctx context.Context, req *paymentv1.PaymentRequest) (*paymentv1.PaymentResponse, error) {
	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}

	payment, err := s.svc.ProcessPayment(req.OrderId, req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "payment processing failed: %v", err)
	}

	return &paymentv1.PaymentResponse{
		PaymentId:   payment.ID,
		OrderId:     payment.OrderID,
		Status:      string(payment.Status),
		ProcessedAt: timestamppb.New(time.Now()),
	}, nil
}
