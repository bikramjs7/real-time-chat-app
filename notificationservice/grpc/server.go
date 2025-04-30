package grpc

import (
	"context"
	"notificationservice/internal/websocket"

	pb "notificationservice/proto"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedNotificationServiceServer
	manager *websocket.WebSocketManager
	logger  *zap.Logger
}

func NewGRPCServer(manager *websocket.WebSocketManager, logger *zap.Logger) *grpc.Server {
	s := grpc.NewServer()
	pb.RegisterNotificationServiceServer(s, &server{manager: manager, logger: logger})
	reflection.Register(s)
	return s
}

func (s *server) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	s.manager.CloseConnection(req.GetUserid())
	s.logger.Info("Logout successfull", zap.String("userid", req.GetUserid()))
	return &pb.LogoutResponse{Success: true}, nil
}

func (s *server) PushMessage(ctx context.Context, req *pb.PushMessageRequest) (*pb.PushMessageResponse, error) {
	s.logger.Info("Push Message gRPC call received")
	msg := websocket.Message{
		Sender:     req.GetSender(),
		Recipients: req.Recipients,
		Message:    req.GetMessage(),
	}
	s.manager.HandleMessages(msg)
	return &pb.PushMessageResponse{Success: true}, nil
}
