package testutil

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"go-digital-wallet/internal/cache"
	"go-digital-wallet/internal/model"
	"go-digital-wallet/internal/repository"
	"go-digital-wallet/internal/service"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// FakeBalanceCache is a controllable cache test double.
type FakeBalanceCache struct {
	Values    map[uuid.UUID]int64
	GetErr    error
	SetErr    error
	DeleteErr error
	PingErr   error
}

// NewFakeBalanceCache creates a test cache.
func NewFakeBalanceCache() *FakeBalanceCache {
	return &FakeBalanceCache{
		Values: make(map[uuid.UUID]int64),
	}
}

// Get returns a cached balance.
func (c *FakeBalanceCache) Get(_ context.Context, walletID uuid.UUID) (int64, bool, error) {
	if c.GetErr != nil {
		return 0, false, c.GetErr
	}

	value, ok := c.Values[walletID]
	return value, ok, nil
}

// Set caches a balance.
func (c *FakeBalanceCache) Set(_ context.Context, walletID uuid.UUID, balance int64) error {
	if c.SetErr != nil {
		return c.SetErr
	}

	c.Values[walletID] = balance
	return nil
}

// Delete evicts a cached balance.
func (c *FakeBalanceCache) Delete(_ context.Context, walletID uuid.UUID) error {
	if c.DeleteErr != nil {
		return c.DeleteErr
	}

	delete(c.Values, walletID)
	return nil
}

// Ping returns the configured ping result.
func (c *FakeBalanceCache) Ping(context.Context) error {
	return c.PingErr
}

var _ cache.BalanceCache = (*FakeBalanceCache)(nil)

// TestEnv wires a lightweight in-memory application environment.
type TestEnv struct {
	DB                 *gorm.DB
	Cache              *FakeBalanceCache
	UserRepo           *repository.UserRepository
	WalletRepo         *repository.WalletRepository
	TransactionRepo    *repository.TransactionRepository
	IdempotencyRepo    *repository.IdempotencyRepository
	OutboxRepo         *repository.OutboxRepository
	UserService        *service.UserService
	WalletService      *service.WalletService
	TransactionService *service.TransactionService
}

// NewTestEnv creates an isolated sqlite-backed environment.
func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if closeErr := sqlDB.Close(); closeErr != nil && !errors.Is(closeErr, gorm.ErrInvalidDB) {
			t.Fatalf("close sqlite: %v", closeErr)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := repository.AutoMigrate(ctx, db); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	if err := repository.SeedSystemAccounts(ctx, db); err != nil {
		t.Fatalf("seed system accounts: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(testingWriter{t: t}, nil))
	cacheStore := NewFakeBalanceCache()

	userRepo := repository.NewUserRepository(db)
	walletRepo := repository.NewWalletRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	idempotencyRepo := repository.NewIdempotencyRepository(db)
	outboxRepo := repository.NewOutboxRepository(db)

	return &TestEnv{
		DB:              db,
		Cache:           cacheStore,
		UserRepo:        userRepo,
		WalletRepo:      walletRepo,
		TransactionRepo: transactionRepo,
		IdempotencyRepo: idempotencyRepo,
		OutboxRepo:      outboxRepo,
		UserService:     service.NewUserService(userRepo, 20, 100),
		WalletService:   service.NewWalletService(logger, userRepo, walletRepo, cacheStore),
		TransactionService: service.NewTransactionService(
			logger,
			db,
			walletRepo,
			transactionRepo,
			idempotencyRepo,
			outboxRepo,
			cacheStore,
			24*time.Hour,
			20,
			100,
		),
	}
}

// SeedUser creates a user fixture.
func (e *TestEnv) SeedUser(t *testing.T, email string) model.User {
	t.Helper()

	return e.SeedUserWithStatus(t, email, model.UserStatusActive)
}

// SeedUserWithStatus creates a user fixture with a specific status.
func (e *TestEnv) SeedUserWithStatus(t *testing.T, email string, status model.UserStatus) model.User {
	t.Helper()

	user := model.User{
		FullName:    "Test User",
		Email:       email,
		PhoneNumber: uuid.NewString()[:12],
		Status:      status,
	}
	if err := e.UserRepo.Create(context.Background(), &user); err != nil {
		t.Fatalf("create user fixture: %v", err)
	}

	return user
}

// SeedWallet creates a wallet fixture with a starting balance.
func (e *TestEnv) SeedWallet(t *testing.T, userID uuid.UUID, balance int64) model.Wallet {
	t.Helper()

	return e.SeedWalletWithStatus(t, userID, balance, model.WalletStatusActive)
}

// SeedWalletWithStatus creates a wallet fixture with a specific status.
func (e *TestEnv) SeedWalletWithStatus(t *testing.T, userID uuid.UUID, balance int64, status model.WalletStatus) model.Wallet {
	t.Helper()

	wallet := model.Wallet{
		UserID:             userID,
		Currency:           model.CurrencyTHB,
		Status:             status,
		BalanceCachedMinor: balance,
	}
	if err := e.WalletRepo.Create(context.Background(), &wallet); err != nil {
		t.Fatalf("create wallet fixture: %v", err)
	}

	return wallet
}

type testingWriter struct {
	t *testing.T
}

func (w testingWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}
