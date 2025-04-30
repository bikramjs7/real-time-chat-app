package main

import (
	"log"
	"net"
	"notificationservice/config"
	"notificationservice/grpc"
	"notificationservice/internal/kafka"
	"notificationservice/internal/routes"
	"notificationservice/internal/websocket"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Can't initialize zap logger: %v", err)
	}
	defer logger.Sync()
	producer := kafka.NewProducer(logger)
	defer producer.Close()

	app := fiber.New()

	manager := websocket.NewWebSocketManager(logger, producer, 8)
	manager.StartWorkerPool()

	routes.Setup(app, manager)

	go func() {
		logger.Info("Starting HTTP server on port 3001")
		if err := app.Listen(config.HTTPPort); err != nil {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	lis, err := net.Listen("tcp", config.GRPCAddress)
	if err != nil {
		logger.Fatal("Failed to listen on port "+config.GRPCAddress, zap.Error(err))
	}
	grpcServer := grpc.NewGRPCServer(manager, logger)
	logger.Info("Starting gRPC server on port " + config.GRPCAddress)
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("Failed to start gRPC server", zap.Error(err))
	}
}
