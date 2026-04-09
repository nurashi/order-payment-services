package grpc

import (
	"time"

	orderv1 "github.com/nurashi/ap2-proto-gen/order/v1"
	"github.com/nurashi/order-service/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderServer struct {
	orderv1.UnimplementedOrderServiceServer
	subscriber domain.OrderSubscriber
}

func NewOrderServer(subscriber domain.OrderSubscriber) *OrderServer {
	return &OrderServer{subscriber: subscriber}
}

func (s *OrderServer) SubscribeToOrderUpdates(req *orderv1.OrderRequest, stream orderv1.OrderService_SubscribeToOrderUpdatesServer) error {
	if req.OrderId == "" {
		return status.Error(codes.InvalidArgument, "order_id is required")
	}

	ch, err := s.subscriber.SubscribeToOrderUpdates(stream.Context(), req.OrderId)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe: %v", err)
	}

	for order := range ch {
		update := &orderv1.OrderStatusUpdate{
			OrderId:   order.ID,
			Status:    string(order.Status),
			UpdatedAt: timestamppb.New(time.Now()),
		}
		if err := stream.Send(update); err != nil {
			return status.Errorf(codes.Unavailable, "failed to send update: %v", err)
		}
	}

	return nil
}
