package message

import (
	"encoding/json"
	"fmt"
	"workerservice/grpcclient"

	"go.uber.org/zap"
)

type MessageHandler struct {
	logger     *zap.Logger
	grpcClient *grpcclient.Client
}

type ChatMessage struct {
	Sender     string   `json:"sender"`
	Recipients []string `json:"recipients"`
	Message    string   `json:"message"`
}

func NewMessageHandler(logger *zap.Logger, addr string) *MessageHandler {
	grpcClient, err := grpcclient.NewClient(addr)
	if err != nil {
		logger.Fatal("failed to create grpc client", zap.Error(err))
	}
	return &MessageHandler{logger: logger, grpcClient: grpcClient}
}

func (h *MessageHandler) HandleMessage(data []byte) error {
	var msg ChatMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}
	_, err := h.grpcClient.PushMessage(msg.Sender, msg.Recipients, msg.Message)
	if err != nil {
		return fmt.Errorf("failed to push message: %w", err)
	} else {
		h.logger.Info("message pushed successfully", zap.String("sender", msg.Sender))
	}
	return nil
}
