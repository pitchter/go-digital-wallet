package repository

import (
	"context"
	"encoding/json"

	"go-digital-wallet/internal/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// IdempotencyRepository encapsulates idempotency persistence.
type IdempotencyRepository struct {
	db *gorm.DB
}

// NewIdempotencyRepository constructs an idempotency repository.
func NewIdempotencyRepository(db *gorm.DB) *IdempotencyRepository {
	return &IdempotencyRepository{db: db}
}

// WithTx returns a repository bound to a transaction handle.
func (r *IdempotencyRepository) WithTx(tx *gorm.DB) *IdempotencyRepository {
	return &IdempotencyRepository{db: tx}
}

// Get fetches an idempotency record.
func (r *IdempotencyRepository) Get(ctx context.Context, key, endpoint string) (*model.IdempotencyKey, error) {
	var record model.IdempotencyKey
	if err := r.db.WithContext(ctx).First(&record, "key = ? AND endpoint = ?", key, endpoint).Error; err != nil {
		return nil, err
	}

	return &record, nil
}

// Create persists an idempotency record.
func (r *IdempotencyRepository) Create(ctx context.Context, record *model.IdempotencyKey) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// SaveResponse persists the final response for replay.
func (r *IdempotencyRepository) SaveResponse(ctx context.Context, recordID uuid.UUID, statusCode int, body any, resourceID *uuid.UUID) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Model(&model.IdempotencyKey{}).
		Where("id = ?", recordID).
		Updates(map[string]any{
			"response_status_code": statusCode,
			"response_body_json":   datatypes.JSON(raw),
			"resource_id":          resourceID,
		}).Error
}
