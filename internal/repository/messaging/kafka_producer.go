package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type kafkaProducer struct {
	writer *kafka.Writer
	topic  string
}

func NewKafkaProducer(brokers []string, topic string) domain.EventProducer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		BatchTimeout:           10 * time.Millisecond,
		Async:                  true,
		AllowAutoTopicCreation: true,
	}

	return &kafkaProducer{
		writer: writer,
		topic:  topic,
	}
}

func (p *kafkaProducer) PublishOrderCreated(ctx context.Context, event domain.OrderCreatedEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal OrderCreatedEvent: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(event.OrderID),
		Value: payload,
		Time:  time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		slog.Error("Failed to dispatch OrderCreated event to Kafka",
			slog.String("order_id", event.OrderID),
			slog.String("error", err.Error()),
		)
		return err
	}

	slog.Info("Successfully dispatched OrderCreated event to Kafka broker",
		slog.String("order_id", event.OrderID),
		slog.String("topic", p.topic),
	)

	return nil
}

func (p *kafkaProducer) Close() error {
	return p.writer.Close()
}
