package grpc

import (
	orderpb "github.com/nurashi/ap2-generated/order/v1"
	"github.com/nurashi/order-service/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderServer struct {
	orderpb.UnimplementedOrderServiceServer
	subscriber domain.OrderSubscriber
}

func NewOrderServer(subscriber domain.OrderSubscriber) *OrderServer {
	return &OrderServer{subscriber: subscriber}
}

func (s *OrderServer) SubscribeToOrderUpdates(req *orderpb.OrderRequest, stream orderpb.OrderService_SubscribeToOrderUpdatesServer) error {
	if req.OrderId == "" {
		return status.Error(codes.InvalidArgument, "order_id is required")
	}

	ch, err := s.subscriber.SubscribeToOrderUpdates(stream.Context(), req.OrderId)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe: %v", err)
	}

	for order := range ch {
		update := &orderpb.OrderStatusUpdate{
			OrderId:   order.ID,
			Status:    string(order.Status),
			UpdatedAt: timestamppb.New(order.UpdatedAt),
		}
		if err := stream.Send(update); err != nil {
			return status.Errorf(codes.Unavailable, "failed to send update: %v", err)
		}
	}

	return nil
}
