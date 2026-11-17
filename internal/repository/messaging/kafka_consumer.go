package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
	"github.com/felix-gnwn/ticketing-engine/internal/middleware"
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

func (c *kafkaConsumer) FetchOrderCreatedEvent(ctx context.Context) (domain.OrderCreatedEvent, context.Context, error) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return domain.OrderCreatedEvent{}, nil, fmt.Errorf("failed to read message from kafka: %w", err)
	}

	var event domain.OrderCreatedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return domain.OrderCreatedEvent{}, nil, fmt.Errorf("failed to unmarshal kafka message payload: %w", err)
	}

	var traceID string
	for _, h := range msg.Headers {
		if h.Key == "trace_id" {
			traceID = string(h.Value)
			break
		}
	}

	msgCtx := context.Background()
	if traceID != "" {
		msgCtx = context.WithValue(msgCtx, middleware.TraceIDKey, traceID)
	}

	return event, msgCtx, nil
}

func (c *kafkaConsumer) Close() error {
	return c.reader.Close()
}
