package repository

import (
	"context"
	"fmt"

	"go-digital-wallet/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OpenPostgres opens the primary application database.
func OpenPostgres(ctx context.Context, dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

// AutoMigrate applies the Gorm schema migration.
func AutoMigrate(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).AutoMigrate(
		&model.User{},
		&model.Wallet{},
		&model.Transaction{},
		&model.LedgerEntry{},
		&model.IdempotencyKey{},
		&model.OutboxEvent{},
	)
}

// SeedSystemAccounts ensures the system settlement account exists.
func SeedSystemAccounts(ctx context.Context, db *gorm.DB) error {
	systemExternalRef := "system-wallet-service"
	systemUser := model.User{
		BaseModel:   model.BaseModel{ID: model.SystemUserID},
		ExternalRef: &systemExternalRef,
		FullName:    "System Wallet Service",
		Email:       "system@wallet-service.local",
		PhoneNumber: "SYSTEM-WALLET",
		Status:      model.UserStatusActive,
	}

	if err := db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&systemUser).Error; err != nil {
		return fmt.Errorf("seed system user: %w", err)
	}

	systemWallet := model.Wallet{
		BaseModel:          model.BaseModel{ID: model.SystemWalletID},
		UserID:             model.SystemUserID,
		Currency:           model.CurrencyTHB,
		Status:             model.WalletStatusActive,
		BalanceCachedMinor: 0,
	}

	if err := db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&systemWallet).Error; err != nil {
		return fmt.Errorf("seed system wallet: %w", err)
	}

	return nil
}
