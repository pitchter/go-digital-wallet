package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// IdempotencyKey stores request replay metadata.
type IdempotencyKey struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Key                string         `gorm:"size:255;not null;uniqueIndex:idx_key_endpoint" json:"key"`
	Endpoint           string         `gorm:"size:255;not null;uniqueIndex:idx_key_endpoint" json:"endpoint"`
	RequestHash        string         `gorm:"size:128;not null" json:"request_hash"`
	ResponseStatusCode *int           `json:"response_status_code,omitempty"`
	ResponseBodyJSON   datatypes.JSON `gorm:"type:jsonb" json:"-"`
	ResourceID         *uuid.UUID     `gorm:"type:uuid" json:"resource_id,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	ExpiresAt          time.Time      `gorm:"index" json:"expires_at"`
}

// BeforeCreate guarantees UUID assignment.
func (i *IdempotencyKey) BeforeCreate(*gorm.DB) error {
	return assignID(&i.ID)
}
