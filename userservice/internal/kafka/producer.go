package kafka

import (
	"context"
	"encoding/json"
	"time"

	"userservice/config"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Producer struct {
	emailWriter *kafka.Writer
	logsWriter  *kafka.Writer
	logger      *zap.Logger
}

func NewProducer(logger *zap.Logger) *Producer {
	emailWriter := &kafka.Writer{
		Addr:     kafka.TCP(config.KafkaBrokers),
		Topic:    config.EmailTopic,
		Balancer: &kafka.LeastBytes{},
	}

	logsWriter := &kafka.Writer{
		Addr:     kafka.TCP(config.KafkaBrokers),
		Topic:    config.LogsTopic,
		Balancer: &kafka.LeastBytes{},
	}

	return &Producer{
		emailWriter: emailWriter,
		logsWriter:  logsWriter,
		logger:      logger,
	}
}

func (p *Producer) SendMessageToEmailTopic(key string, value interface{}) error {
	return p.sendMessage(p.emailWriter, key, value)
}

func (p *Producer) SendMessageToLogsTopic(key string, value interface{}) error {
	return p.sendMessage(p.logsWriter, key, value)
}

func (p *Producer) sendMessage(writer *kafka.Writer, key string, value interface{}) error {
	message, err := json.Marshal(value)
	if err != nil {
		return err
	}

	err = writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(key),
			Value: message,
			Time:  time.Now(),
		},
	)
	if err != nil {
		p.logger.Error("Failed to send message", zap.Error(err))
		return err
	}

	p.logger.Info("Message sent successfully", zap.String("key", key))
	return nil
}

func (p *Producer) Close() error {
	if err := p.emailWriter.Close(); err != nil {
		return err
	}
	if err := p.logsWriter.Close(); err != nil {
		return err
	}
	return nil
}
