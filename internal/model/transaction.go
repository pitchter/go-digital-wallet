package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TransactionType categorizes wallet transactions.
type TransactionType string

// TransactionStatus tracks processing state.
type TransactionStatus string

const (
	// TransactionTypeTopUp credits a wallet from the system settlement account.
	TransactionTypeTopUp TransactionType = "topup"
	// TransactionTypeTransfer moves funds between wallets.
	TransactionTypeTransfer TransactionType = "transfer"
	// TransactionTypeReversal compensates a completed transaction.
	TransactionTypeReversal TransactionType = "reversal"
)

const (
	// TransactionStatusPending indicates a not-yet-finalized transaction.
	TransactionStatusPending TransactionStatus = "pending"
	// TransactionStatusCompleted indicates a posted transaction.
	TransactionStatusCompleted TransactionStatus = "completed"
	// TransactionStatusFailed indicates a failed transaction.
	TransactionStatusFailed TransactionStatus = "failed"
	// TransactionStatusReversed indicates the original transaction was reversed.
	TransactionStatusReversed TransactionStatus = "reversed"
)

// Transaction records immutable money movement.
type Transaction struct {
	BaseModel
	ReferenceCode        string            `gorm:"size:64;not null;uniqueIndex" json:"reference_code"`
	Type                 TransactionType   `gorm:"size:16;not null" json:"type"`
	Status               TransactionStatus `gorm:"size:16;not null" json:"status"`
	SourceWalletID       *uuid.UUID        `gorm:"type:uuid;index" json:"source_wallet_id,omitempty"`
	DestinationWalletID  *uuid.UUID        `gorm:"type:uuid;index" json:"destination_wallet_id,omitempty"`
	AmountMinor          int64             `gorm:"not null" json:"amount_minor"`
	Currency             string            `gorm:"size:3;not null;default:THB" json:"currency"`
	IdempotencyKey       *string           `gorm:"size:255;index" json:"idempotency_key,omitempty"`
	RelatedTransactionID *uuid.UUID        `gorm:"type:uuid;index" json:"related_transaction_id,omitempty"`
	MetadataJSON         datatypes.JSONMap `gorm:"type:jsonb" json:"metadata,omitempty"`
}

// BeforeCreate guarantees defaults and UUID assignment.
func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	if err := t.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}

	if t.Currency == "" {
		t.Currency = CurrencyTHB
	}
	if t.Status == "" {
		t.Status = TransactionStatusPending
	}

	return nil
}

// LedgerEntryType captures debit or credit.
type LedgerEntryType string

const (
	// LedgerEntryTypeDebit reduces a wallet balance.
	LedgerEntryTypeDebit LedgerEntryType = "debit"
	// LedgerEntryTypeCredit increases a wallet balance.
	LedgerEntryTypeCredit LedgerEntryType = "credit"
)

// LedgerEntry is an append-only line for a transaction.
type LedgerEntry struct {
	ID            uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	TransactionID uuid.UUID       `gorm:"type:uuid;not null;index" json:"transaction_id"`
	WalletID      uuid.UUID       `gorm:"type:uuid;not null;index" json:"wallet_id"`
	EntryType     LedgerEntryType `gorm:"size:16;not null" json:"entry_type"`
	AmountMinor   int64           `gorm:"not null" json:"amount_minor"`
	CreatedAt     time.Time       `json:"created_at"`
}

// BeforeCreate guarantees UUID assignment.
func (l *LedgerEntry) BeforeCreate(*gorm.DB) error {
	return assignID(&l.ID)
}
