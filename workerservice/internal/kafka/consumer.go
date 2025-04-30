package kafka

import (
	"context"
	"sync"

	"workerservice/config"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// MessageProcessor is a function type that processes Kafka messages
type MessageProcessor func(message []byte) error

type Consumer struct {
	readers    []*kafka.Reader
	handlers   map[string]MessageProcessor
	numWorkers int
	wg         sync.WaitGroup
	jobs       chan kafka.Message
	logger     *zap.Logger
}

func NewConsumer(logger *zap.Logger) (*Consumer, error) {
	topics := []string{config.EmailTopic, config.LogsTopic, config.MessageTopic}
	readers := make([]*kafka.Reader, len(topics))
	for i, topic := range topics {
		readers[i] = kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{config.KafkaBrokers},
			GroupID: config.KafkaGroupID,
			Topic:   topic,
		})
	}

	return &Consumer{
		readers:    readers,
		handlers:   make(map[string]MessageProcessor),
		numWorkers: config.KafkaWorkers,
		jobs:       make(chan kafka.Message, 1000),
		logger:     logger,
	}, nil
}

func (c *Consumer) RegisterHandler(topic string, handler MessageProcessor) {
	c.handlers[topic] = handler
}

func (c *Consumer) Start(ctx context.Context) {
	for i := 0; i < c.numWorkers; i++ {
		c.wg.Add(1)
		go c.worker(i, c.jobs)
	}

	for _, reader := range c.readers {
		go func(reader *kafka.Reader) {
			for {
				m, err := reader.ReadMessage(ctx)
				if err != nil {
					if err == context.Canceled || err == context.DeadlineExceeded || err == kafka.ErrGroupClosed {
						c.logger.Info("Kafka reader context canceled or closed", zap.Error(err))
						return
					}
					//	c.logger.Error("Error reading message", zap.Error(err))
					continue
				}
				c.jobs <- m
			}
		}(reader)
	}

	c.logger.Info("Kafka consumer started")
}

func (c *Consumer) worker(id int, jobs <-chan kafka.Message) {
	defer c.wg.Done()
	c.logger.Info("Worker started", zap.Int("worker_id", id))

	for msg := range jobs {
		c.logger.Info("Processing message", zap.Int("worker_id", id), zap.String("topic", msg.Topic), zap.String("value", string(msg.Value)))
		if handler, ok := c.handlers[msg.Topic]; ok {
			if err := handler(msg.Value); err != nil {
				c.logger.Error("Error processing message", zap.Int("worker_id", id), zap.Error(err))
			}
		}
	}

	c.logger.Info("Worker stopped", zap.Int("worker_id", id))
}

func (c *Consumer) Close() error {
	close(c.jobs)
	c.wg.Wait()
	for _, reader := range c.readers {
		if err := reader.Close(); err != nil {
			return err
		}
	}
	return nil
}
