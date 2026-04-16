package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WalletStatus is the lifecycle state of a wallet.
type WalletStatus string

const (
	// WalletStatusActive allows money movement.
	WalletStatusActive WalletStatus = "active"
	// WalletStatusSuspended blocks money movement.
	WalletStatusSuspended WalletStatus = "suspended"
	// WalletStatusClosed blocks money movement permanently.
	WalletStatusClosed WalletStatus = "closed"
)

// Wallet stores balances in minor units.
type Wallet struct {
	BaseModel
	UserID             uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	Currency           string         `gorm:"size:3;not null;default:THB" json:"currency"`
	Status             WalletStatus   `gorm:"size:16;not null;default:active" json:"status"`
	BalanceCachedMinor int64          `gorm:"not null;default:0" json:"balance_minor"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate guarantees defaults and UUID assignment.
func (w *Wallet) BeforeCreate(tx *gorm.DB) error {
	if err := w.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}

	if w.Currency == "" {
		w.Currency = CurrencyTHB
	}
	if w.Status == "" {
		w.Status = WalletStatusActive
	}

	return nil
}

// IsActive returns true when the wallet can accept money movement.
func (w Wallet) IsActive() bool {
	return w.Status == WalletStatusActive
}
