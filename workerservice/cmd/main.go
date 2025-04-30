package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"workerservice/config"
	email "workerservice/internal/email"
	"workerservice/internal/kafka"
	logs "workerservice/internal/logs"
	"workerservice/internal/message"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize zap logger: %v", err)
	}
	defer logger.Sync()

	if err := kafka.CreateTopics(logger); err != nil {
		logger.Fatal("Failed to create Kafka topics", zap.Error(err))
	}

	logRepo, err := logs.NewLogRepository(logger)
	if err != nil {
		logger.Fatal("Failed to initialize log repository", zap.Error(err))
	}
	defer logRepo.Close()

	emailSender := email.NewEmailSender(logger)
	msgHandler := message.NewMessageHandler(logger, config.GRPCAddress)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer, err := kafka.NewConsumer(logger)
	if err != nil {
		logger.Fatal("Failed to create consumer", zap.Error(err))
	}

	consumer.RegisterHandler(config.EmailTopic, emailSender.SendRegistrationEmail)
	consumer.RegisterHandler(config.LogsTopic, logRepo.StoreLog)
	consumer.RegisterHandler(config.MessageTopic, msgHandler.HandleMessage)

	consumer.Start(ctx)

	logger.Info("Worker service started. Press Ctrl+C to exit.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down...")
	cancel()
	if err := consumer.Close(); err != nil {
		logger.Error("Failed to close Kafka consumer", zap.Error(err))
	}
	logger.Info("Worker service stopped gracefully")
}
