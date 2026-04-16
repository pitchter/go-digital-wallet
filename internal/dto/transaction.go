package dto

import (
	"time"

	"go-digital-wallet/internal/model"

	"github.com/google/uuid"
)

// TopUpRequest is the top-up payload.
type TopUpRequest struct {
	WalletID    uuid.UUID      `json:"wallet_id" binding:"required"`
	AmountMinor int64          `json:"amount_minor" binding:"required,gt=0"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// TransferRequest is the transfer payload.
type TransferRequest struct {
	SourceWalletID      uuid.UUID      `json:"source_wallet_id" binding:"required"`
	DestinationWalletID uuid.UUID      `json:"destination_wallet_id" binding:"required"`
	AmountMinor         int64          `json:"amount_minor" binding:"required,gt=0"`
	Metadata            map[string]any `json:"metadata,omitempty"`
}

// TransactionResponse is the immutable transaction API payload.
type TransactionResponse struct {
	ID                   uuid.UUID               `json:"id"`
	ReferenceCode        string                  `json:"reference_code"`
	Type                 model.TransactionType   `json:"type"`
	Status               model.TransactionStatus `json:"status"`
	SourceWalletID       *uuid.UUID              `json:"source_wallet_id,omitempty"`
	DestinationWalletID  *uuid.UUID              `json:"destination_wallet_id,omitempty"`
	AmountMinor          int64                   `json:"amount_minor"`
	Currency             string                  `json:"currency"`
	IdempotencyKey       *string                 `json:"idempotency_key,omitempty"`
	RelatedTransactionID *uuid.UUID              `json:"related_transaction_id,omitempty"`
	Metadata             map[string]any          `json:"metadata,omitempty"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            time.Time               `json:"updated_at"`
}

// TransactionFromModel converts a model into an API payload.
func TransactionFromModel(tx model.Transaction) TransactionResponse {
	return TransactionResponse{
		ID:                   tx.ID,
		ReferenceCode:        tx.ReferenceCode,
		Type:                 tx.Type,
		Status:               tx.Status,
		SourceWalletID:       tx.SourceWalletID,
		DestinationWalletID:  tx.DestinationWalletID,
		AmountMinor:          tx.AmountMinor,
		Currency:             tx.Currency,
		IdempotencyKey:       tx.IdempotencyKey,
		RelatedTransactionID: tx.RelatedTransactionID,
		Metadata:             map[string]any(tx.MetadataJSON),
		CreatedAt:            tx.CreatedAt,
		UpdatedAt:            tx.UpdatedAt,
	}
}
