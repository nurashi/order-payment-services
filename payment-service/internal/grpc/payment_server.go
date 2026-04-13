package grpc

import (
	"context"

	paymentpb "github.com/nurashi/ap2-generated/payment/v1"
	"github.com/nurashi/payment-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PaymentServer struct {
	paymentpb.UnimplementedPaymentServiceServer
	svc service.PaymentService
}

func NewPaymentServer(svc service.PaymentService) *PaymentServer {
	return &PaymentServer{svc: svc}
}

func (s *PaymentServer) ProcessPayment(ctx context.Context, req *paymentpb.PaymentRequest) (*paymentpb.PaymentResponse, error) {
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

	return &paymentpb.PaymentResponse{
		PaymentId:   payment.ID,
		OrderId:     payment.OrderID,
		Status:      string(payment.Status),
		ProcessedAt: timestamppb.New(payment.UpdatedAt),
	}, nil
}
