package service_test

import (
	"context"
	"errors"
	"testing"

	"go-digital-wallet/internal/dto"
	"go-digital-wallet/internal/model"
	"go-digital-wallet/internal/testutil"

	"github.com/google/uuid"
)

func TestWalletServiceCreateSuccess(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "wallet-create@example.com")

	response, err := env.WalletService.Create(context.Background(), dto.CreateWalletRequest{
		UserID: user.ID,
	})
	if err != nil {
		t.Fatalf("create wallet: %v", err)
	}

	if response.ID == uuid.Nil {
		t.Fatal("expected wallet ID")
	}
	if response.Currency != model.CurrencyTHB {
		t.Fatalf("expected currency %s, got %s", model.CurrencyTHB, response.Currency)
	}
	if response.Status != model.WalletStatusActive {
		t.Fatalf("expected active status, got %s", response.Status)
	}
	if response.BalanceMinor != 0 {
		t.Fatalf("expected zero balance, got %d", response.BalanceMinor)
	}
}

func TestWalletServiceCreateInactiveUserValidation(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUserWithStatus(t, "wallet-inactive-user@example.com", model.UserStatusInactive)

	_, err := env.WalletService.Create(context.Background(), dto.CreateWalletRequest{
		UserID: user.ID,
	})
	assertAppErrorCode(t, err, "validation_error")
}

func TestWalletServiceCreateDuplicateConflict(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "wallet-duplicate@example.com")

	if _, err := env.WalletService.Create(context.Background(), dto.CreateWalletRequest{UserID: user.ID}); err != nil {
		t.Fatalf("create first wallet: %v", err)
	}

	_, err := env.WalletService.Create(context.Background(), dto.CreateWalletRequest{UserID: user.ID})
	assertAppErrorCode(t, err, "resource_conflict")
}

func TestWalletServiceGetSuccess(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "wallet-get@example.com")
	wallet := env.SeedWallet(t, user.ID, 1234)

	response, err := env.WalletService.Get(context.Background(), wallet.ID)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}

	if response.ID != wallet.ID {
		t.Fatalf("expected wallet ID %s, got %s", wallet.ID, response.ID)
	}
	if response.BalanceMinor != 1234 {
		t.Fatalf("expected balance 1234, got %d", response.BalanceMinor)
	}
}

func TestWalletServiceGetMissingWallet(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	_, err := env.WalletService.Get(context.Background(), uuid.New())
	assertAppErrorCode(t, err, "resource_not_found")
}

func TestWalletServiceGetBalanceUsesRedisCacheHit(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "wallet-cache-hit@example.com")
	wallet := env.SeedWallet(t, user.ID, 1234)
	env.Cache.Values[wallet.ID] = 4321

	response, err := env.WalletService.GetBalance(context.Background(), wallet.ID)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}

	if response.BalanceMinor != 4321 {
		t.Fatalf("expected cached balance 4321, got %d", response.BalanceMinor)
	}
	if response.Source != "redis" {
		t.Fatalf("expected redis source, got %s", response.Source)
	}
}

func TestWalletServiceGetBalanceFallsBackWhenCacheFails(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "wallet-cache@example.com")
	wallet := env.SeedWallet(t, user.ID, 1234)
	env.Cache.GetErr = errors.New("redis down")

	response, err := env.WalletService.GetBalance(context.Background(), wallet.ID)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}

	if response.BalanceMinor != 1234 {
		t.Fatalf("expected balance 1234, got %d", response.BalanceMinor)
	}
	if response.Source != "database" {
		t.Fatalf("expected database source, got %s", response.Source)
	}
}

func TestWalletServiceGetBalanceMissingWallet(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	_, err := env.WalletService.GetBalance(context.Background(), uuid.New())
	assertAppErrorCode(t, err, "resource_not_found")
}

func TestWalletServiceUpdateStatusSuccess(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "wallet-status@example.com")
	wallet := env.SeedWallet(t, user.ID, 0)

	response, err := env.WalletService.UpdateStatus(context.Background(), wallet.ID, model.WalletStatusSuspended)
	if err != nil {
		t.Fatalf("update wallet status: %v", err)
	}

	if response.Status != model.WalletStatusSuspended {
		t.Fatalf("expected suspended status, got %s", response.Status)
	}

	stored, err := env.WalletRepo.GetByID(context.Background(), wallet.ID)
	if err != nil {
		t.Fatalf("load wallet: %v", err)
	}
	if stored.Status != model.WalletStatusSuspended {
		t.Fatalf("expected stored wallet to be suspended, got %s", stored.Status)
	}
}

func TestWalletServiceUpdateStatusMissingWallet(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	_, err := env.WalletService.UpdateStatus(context.Background(), uuid.New(), model.WalletStatusClosed)
	assertAppErrorCode(t, err, "resource_not_found")
}
