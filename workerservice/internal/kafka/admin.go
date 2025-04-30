package kafka

import (
	"context"

	"workerservice/config"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

func CreateTopics(logger *zap.Logger) error {
	topics := []string{config.EmailTopic, config.LogsTopic, config.MessageTopic}
	for _, topic := range topics {
		conn, err := kafka.DialLeader(context.Background(), "tcp", config.KafkaBrokers, topic, 0)
		if err != nil {
			logger.Fatal("Error connecting to Kafka leader", zap.String("topic", topic), zap.Error(err))
			continue
		}
		defer conn.Close()

		err = conn.CreateTopics(kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     3,
			ReplicationFactor: 1,
		})
		if err != nil {
			logger.Error("Error creating topic", zap.String("topic", topic), zap.Error(err))
		} else {
			logger.Info("Topic created successfully", zap.String("topic", topic))
		}
	}
	return nil
}
