package repository

import (
	"context"

	"go-digital-wallet/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TransactionFilters scopes transaction listing queries.
type TransactionFilters struct {
	WalletID *uuid.UUID
	Type     *model.TransactionType
	Status   *model.TransactionStatus
}

// TransactionRepository encapsulates transaction persistence.
type TransactionRepository struct {
	db *gorm.DB
}

// NewTransactionRepository constructs a transaction repository.
func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// WithTx returns a repository bound to a transaction handle.
func (r *TransactionRepository) WithTx(tx *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: tx}
}

// Create persists a transaction.
func (r *TransactionRepository) Create(ctx context.Context, txModel *model.Transaction) error {
	return r.db.WithContext(ctx).Create(txModel).Error
}

// CreateLedgerEntries persists append-only ledger entries.
func (r *TransactionRepository) CreateLedgerEntries(ctx context.Context, entries []model.LedgerEntry) error {
	if len(entries) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Create(&entries).Error
}

// GetByID fetches a transaction by ID.
func (r *TransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {
	var txModel model.Transaction
	if err := r.db.WithContext(ctx).First(&txModel, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &txModel, nil
}

// GetByIDForUpdate fetches a transaction with row-level locking.
func (r *TransactionRepository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {
	var txModel model.Transaction
	if err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&txModel, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &txModel, nil
}

// Update persists transaction changes.
func (r *TransactionRepository) Update(ctx context.Context, txModel *model.Transaction) error {
	return r.db.WithContext(ctx).Save(txModel).Error
}

// List returns paginated transactions with optional filters.
func (r *TransactionRepository) List(ctx context.Context, page, limit int, filters TransactionFilters) ([]model.Transaction, int64, error) {
	var transactions []model.Transaction
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Transaction{})
	if filters.WalletID != nil {
		query = query.Where("source_wallet_id = ? OR destination_wallet_id = ?", *filters.WalletID, *filters.WalletID)
	}
	if filters.Type != nil {
		query = query.Where("type = ?", *filters.Type)
	}
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}
