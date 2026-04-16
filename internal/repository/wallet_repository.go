package repository

import (
	"context"
	"sort"

	"go-digital-wallet/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WalletRepository encapsulates wallet persistence.
type WalletRepository struct {
	db *gorm.DB
}

// NewWalletRepository constructs a wallet repository.
func NewWalletRepository(db *gorm.DB) *WalletRepository {
	return &WalletRepository{db: db}
}

// WithTx returns a repository bound to a transaction handle.
func (r *WalletRepository) WithTx(tx *gorm.DB) *WalletRepository {
	return &WalletRepository{db: tx}
}

// Create persists a wallet.
func (r *WalletRepository) Create(ctx context.Context, wallet *model.Wallet) error {
	return r.db.WithContext(ctx).Create(wallet).Error
}

// GetByID fetches a wallet by ID.
func (r *WalletRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Wallet, error) {
	var wallet model.Wallet
	if err := r.db.WithContext(ctx).First(&wallet, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &wallet, nil
}

// GetByUserID fetches a wallet by user ID.
func (r *WalletRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Wallet, error) {
	var wallet model.Wallet
	if err := r.db.WithContext(ctx).First(&wallet, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}

	return &wallet, nil
}

// GetByIDsForUpdate fetches wallets with row-level locking in a deterministic order.
func (r *WalletRepository) GetByIDsForUpdate(ctx context.Context, ids ...uuid.UUID) ([]model.Wallet, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	sort.Slice(ids, func(i, j int) bool {
		return ids[i].String() < ids[j].String()
	})

	var wallets []model.Wallet
	if err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ?", ids).
		Order("id asc").
		Find(&wallets).Error; err != nil {
		return nil, err
	}

	return wallets, nil
}

// Update persists wallet changes.
func (r *WalletRepository) Update(ctx context.Context, wallet *model.Wallet) error {
	return r.db.WithContext(ctx).Save(wallet).Error
}
