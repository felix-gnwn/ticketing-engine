package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type kafkaConsumer struct {
	reader *kafka.Reader
}

func NewKafkaConsumer(brokers []string, topic, groupID string, minBytes, maxBytes int) domain.EventConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: minBytes,
		MaxBytes: maxBytes,
	})

	return &kafkaConsumer{
		reader: reader,
	}
}

func (c *kafkaConsumer) FetchOrderCreatedEvent(ctx context.Context) (domain.OrderCreatedEvent, error) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return domain.OrderCreatedEvent{}, fmt.Errorf("failed to read message from kafka: %w", err)
	}

	var event domain.OrderCreatedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return domain.OrderCreatedEvent{}, fmt.Errorf("failed to unmarshal kafka message payload: %w", err)
	}

	return event, nil
}

func (c *kafkaConsumer) Close() error {
	return c.reader.Close()
}
