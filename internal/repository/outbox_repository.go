package repository

import (
	"context"
	"time"

	"go-digital-wallet/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OutboxRepository encapsulates outbox persistence.
type OutboxRepository struct {
	db *gorm.DB
}

// NewOutboxRepository constructs an outbox repository.
func NewOutboxRepository(db *gorm.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

// WithTx returns a repository bound to a transaction handle.
func (r *OutboxRepository) WithTx(tx *gorm.DB) *OutboxRepository {
	return &OutboxRepository{db: tx}
}

// Create persists an outbox event.
func (r *OutboxRepository) Create(ctx context.Context, event *model.OutboxEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// FetchPending returns the next batch of publishable events.
func (r *OutboxRepository) FetchPending(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	var events []model.OutboxEvent
	if err := r.db.WithContext(ctx).
		Where("status = ?", model.OutboxStatusPending).
		Order("created_at asc").
		Limit(limit).
		Find(&events).Error; err != nil {
		return nil, err
	}

	return events, nil
}

// MarkPublished marks an outbox event as published.
func (r *OutboxRepository) MarkPublished(ctx context.Context, id uuid.UUID, publishedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.OutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       model.OutboxStatusPublished,
			"published_at": publishedAt,
			"last_error":   nil,
		}).Error
}

// RecordFailure increments failure metadata while keeping the event retryable.
func (r *OutboxRepository) RecordFailure(ctx context.Context, id uuid.UUID, errMessage string) error {
	return r.db.WithContext(ctx).
		Model(&model.OutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"retry_count": gorm.Expr("retry_count + 1"),
			"last_error":  errMessage,
			"status":      model.OutboxStatusPending,
		}).Error
}
