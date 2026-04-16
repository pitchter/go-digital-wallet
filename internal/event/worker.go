package event

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go-digital-wallet/internal/repository"
)

// Worker publishes pending outbox events to Redis Streams.
type Worker struct {
	logger    *slog.Logger
	repo      *repository.OutboxRepository
	publisher StreamPublisher
	batchSize int
	stream    string
}

// NewWorker constructs an outbox worker.
func NewWorker(
	logger *slog.Logger,
	repo *repository.OutboxRepository,
	publisher StreamPublisher,
	batchSize int,
	stream string,
) *Worker {
	return &Worker{
		logger:    logger,
		repo:      repo,
		publisher: publisher,
		batchSize: batchSize,
		stream:    stream,
	}
}

// Start launches the worker loop.
func (w *Worker) Start(ctx context.Context, interval time.Duration) {
	go w.run(ctx, interval)
}

func (w *Worker) run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := w.PublishPending(ctx); err != nil && ctx.Err() == nil {
			w.logger.Error("publish pending outbox events", "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// PublishPending publishes a single batch of pending events.
func (w *Worker) PublishPending(ctx context.Context) error {
	events, err := w.repo.FetchPending(ctx, w.batchSize)
	if err != nil {
		return fmt.Errorf("fetch pending outbox events: %w", err)
	}

	for _, evt := range events {
		payload, err := ParsePayload(evt.PayloadJSON)
		if err != nil {
			recordErr := w.repo.RecordFailure(ctx, evt.ID, fmt.Sprintf("decode payload: %v", err))
			if recordErr != nil {
				return fmt.Errorf("decode payload and record failure: %w", recordErr)
			}
			continue
		}

		if err := w.publisher.Publish(ctx, w.stream, payload.ToStreamValues()); err != nil {
			recordErr := w.repo.RecordFailure(ctx, evt.ID, err.Error())
			if recordErr != nil {
				return fmt.Errorf("publish event and record failure: %w", recordErr)
			}
			continue
		}

		if err := w.repo.MarkPublished(ctx, evt.ID, time.Now().UTC()); err != nil {
			return fmt.Errorf("mark published: %w", err)
		}
	}

	return nil
}
