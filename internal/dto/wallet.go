package dto

import (
	"time"

	"go-digital-wallet/internal/model"

	"github.com/google/uuid"
)

// CreateWalletRequest is the POST /wallets payload.
type CreateWalletRequest struct {
	UserID   uuid.UUID `json:"user_id" binding:"required"`
	Currency string    `json:"currency,omitempty" binding:"omitempty,oneof=THB"`
}

// UpdateWalletStatusRequest updates a wallet lifecycle state.
type UpdateWalletStatusRequest struct {
	Status model.WalletStatus `json:"status" binding:"required,oneof=active suspended closed"`
}

// WalletResponse is the wallet API payload.
type WalletResponse struct {
	ID           uuid.UUID          `json:"id"`
	UserID       uuid.UUID          `json:"user_id"`
	Currency     string             `json:"currency"`
	Status       model.WalletStatus `json:"status"`
	BalanceMinor int64              `json:"balance_minor"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// BalanceResponse is the wallet balance payload.
type BalanceResponse struct {
	WalletID     uuid.UUID `json:"wallet_id"`
	Currency     string    `json:"currency"`
	BalanceMinor int64     `json:"balance_minor"`
	Source       string    `json:"source"`
}

// WalletFromModel converts a model into an API payload.
func WalletFromModel(wallet model.Wallet) WalletResponse {
	return WalletResponse{
		ID:           wallet.ID,
		UserID:       wallet.UserID,
		Currency:     wallet.Currency,
		Status:       wallet.Status,
		BalanceMinor: wallet.BalanceCachedMinor,
		CreatedAt:    wallet.CreatedAt,
		UpdatedAt:    wallet.UpdatedAt,
	}
}
