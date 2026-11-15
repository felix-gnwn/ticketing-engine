package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/felix-gnwn/ticketing-engine/internal/domain"
)

type OrderProcessor struct {
	consumer    domain.EventConsumer
	jobChannel  chan domain.OrderCreatedEvent
	workerCount int
	wg          sync.WaitGroup
}

func NewOrderProcessor(consumer domain.EventConsumer, workerCount, bufferSize int) *OrderProcessor {
	return &OrderProcessor{
		consumer:    consumer,
		jobChannel:  make(chan domain.OrderCreatedEvent, bufferSize),
		workerCount: workerCount,
	}
}

func (p *OrderProcessor) Start(ctx context.Context) {
	slog.Info("Starting Order Event Worker Pool", slog.Int("worker_count", p.workerCount))

	for i := 0; i <= p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	go func() {
		defer close(p.jobChannel)

		for {
			select {
			case <-ctx.Done():
				slog.Info("Consumer fetch loop stopping due to context cancellation")
				return
			default:
				event, err := p.consumer.FetchOrderCreatedEvent(ctx)
				if err != nil {
					if ctx.Err() != nil {
						slog.Info("Consumer loop exiting on context cancellation")
						return // Graceful shutdown
					}

					slog.Error("Error fetching event from Kafka, retrying in 1s...", slog.String("error", err.Error()))
					time.Sleep(1 * time.Second)
					continue
				}

				select {
				case p.jobChannel <- event:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
}

func (p *OrderProcessor) worker(ctx context.Context, workerID int) {
	defer p.wg.Done()
	slog.Info("Worker started", slog.Int("worker_id", workerID))

	for event := range p.jobChannel {
		slog.Info("Worker processing OrderCreatedEvent",
			slog.Int("worker_id", workerID),
			slog.String("order_id", event.OrderID),
			slog.String("user_id", event.UserID),
		)

		// Simulate Background Job: Generate PDF Ticket & Send Email Notification
		p.processBackgroundJob(event)
	}

	slog.Info("Worker stopped gracefully", slog.Int("worker_id", workerID))
}

func (p *OrderProcessor) processBackgroundJob(event domain.OrderCreatedEvent) {
	// Simulate PDF generation & notification dispatch latency
	time.Sleep(200 * time.Millisecond)

	slog.Info("SUCCESS: Background job executed (PDF generated & Email sent)",
		slog.String("order_id", event.OrderID),
		slog.String("ticket_id", event.TicketID),
		slog.Float64("total_price", event.TotalPrice),
	)
}

func (p *OrderProcessor) Stop() {
	p.wg.Wait()
	slog.Info("All background workers shut down cleanly")
}
