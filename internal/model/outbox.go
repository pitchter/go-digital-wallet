package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// OutboxStatus captures publication progress.
type OutboxStatus string

const (
	// OutboxStatusPending indicates an event not yet published.
	OutboxStatusPending OutboxStatus = "pending"
	// OutboxStatusPublished indicates an event published successfully.
	OutboxStatusPublished OutboxStatus = "published"
	// OutboxStatusFailed is reserved for manual inspection.
	OutboxStatusFailed OutboxStatus = "failed"
)

// OutboxEvent persists an async publishable event.
type OutboxEvent struct {
	BaseModel
	AggregateType string         `gorm:"size:64;not null" json:"aggregate_type"`
	AggregateID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"aggregate_id"`
	EventType     string         `gorm:"size:64;not null" json:"event_type"`
	PayloadJSON   datatypes.JSON `gorm:"type:jsonb;not null" json:"payload_json"`
	Status        OutboxStatus   `gorm:"size:16;not null;index;default:pending" json:"status"`
	PublishedAt   *time.Time     `json:"published_at,omitempty"`
	RetryCount    int            `gorm:"not null;default:0" json:"retry_count"`
	LastError     *string        `gorm:"type:text" json:"last_error,omitempty"`
}

// BeforeCreate guarantees defaults and UUID assignment.
func (o *OutboxEvent) BeforeCreate(tx *gorm.DB) error {
	if err := o.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}

	if o.Status == "" {
		o.Status = OutboxStatusPending
	}

	return nil
}
