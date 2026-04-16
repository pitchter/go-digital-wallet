package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	// CurrencyTHB is the only supported currency for the MVP.
	CurrencyTHB = "THB"
)

// System account constants back top-up double-entry bookkeeping.
var (
	SystemUserID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	SystemWalletID = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

// assignID ensures a UUID exists before persistence.
func assignID(id *uuid.UUID) error {
	if *id == uuid.Nil {
		*id = uuid.New()
	}

	return nil
}

// BaseModel carries the common ID and timestamp fields.
type BaseModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate guarantees UUID assignment.
func (b *BaseModel) BeforeCreate(*gorm.DB) error {
	return assignID(&b.ID)
}
