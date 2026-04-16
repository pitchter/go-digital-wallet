package service_test

import (
	"context"
	"sync"
	"testing"

	"go-digital-wallet/internal/dto"
	"go-digital-wallet/internal/model"
	"go-digital-wallet/internal/repository"
	"go-digital-wallet/internal/testutil"

	"github.com/google/uuid"
)

func TestTransactionServiceTopUpCreatesOutboxAndLedger(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "topup@example.com")
	wallet := env.SeedWallet(t, user.ID, 0)

	response := mustTopUp(t, env, wallet.ID, 10000, "topup-key")

	if response.AmountMinor != 10000 {
		t.Fatalf("expected amount 10000, got %d", response.AmountMinor)
	}

	assertWalletBalance(t, env, wallet.ID, 10000)
	assertCount(t, env, &model.LedgerEntry{}, "transaction_id = ?", response.ID, 2)
	assertCount(t, env, &model.OutboxEvent{}, "aggregate_id = ?", response.ID, 1)
}

func TestTransactionServiceTopUpMissingWallet(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)

	_, _, err := env.TransactionService.TopUp(context.Background(), dto.TopUpRequest{
		WalletID:    uuid.New(),
		AmountMinor: 10000,
	}, "missing-wallet-key")
	assertAppErrorCode(t, err, "resource_not_found")
}

func TestTransactionServiceTopUpInactiveWallet(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "topup-inactive@example.com")
	wallet := env.SeedWalletWithStatus(t, user.ID, 0, model.WalletStatusSuspended)

	_, _, err := env.TransactionService.TopUp(context.Background(), dto.TopUpRequest{
		WalletID:    wallet.ID,
		AmountMinor: 10000,
	}, "inactive-wallet-key")
	assertAppErrorCode(t, err, "wallet_not_active")
}

func TestTransactionServiceTopUpIdempotencyConflict(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "topup-idem@example.com")
	wallet := env.SeedWallet(t, user.ID, 0)

	mustTopUp(t, env, wallet.ID, 10000, "topup-idem-key")

	_, _, err := env.TransactionService.TopUp(context.Background(), dto.TopUpRequest{
		WalletID:    wallet.ID,
		AmountMinor: 12000,
	}, "topup-idem-key")
	assertAppErrorCode(t, err, "idempotency_conflict")
	assertCount(t, env, &model.Transaction{}, "type = ?", model.TransactionTypeTopUp, 1)
}

func TestTransactionServiceTransferSuccess(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	sourceUser := env.SeedUser(t, "transfer-source@example.com")
	destinationUser := env.SeedUser(t, "transfer-destination@example.com")
	sourceWallet := env.SeedWallet(t, sourceUser.ID, 10000)
	destinationWallet := env.SeedWallet(t, destinationUser.ID, 0)

	response := mustTransfer(t, env, sourceWallet.ID, destinationWallet.ID, 2500, "transfer-success-key")

	if response.AmountMinor != 2500 {
		t.Fatalf("expected amount 2500, got %d", response.AmountMinor)
	}

	assertWalletBalance(t, env, sourceWallet.ID, 7500)
	assertWalletBalance(t, env, destinationWallet.ID, 2500)
	assertCount(t, env, &model.LedgerEntry{}, "transaction_id = ?", response.ID, 2)
	assertCount(t, env, &model.OutboxEvent{}, "aggregate_id = ?", response.ID, 1)
}

func TestTransactionServiceTransferInsufficientFunds(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	sourceUser := env.SeedUser(t, "source-insufficient@example.com")
	destinationUser := env.SeedUser(t, "destination-insufficient@example.com")
	sourceWallet := env.SeedWallet(t, sourceUser.ID, 1000)
	destinationWallet := env.SeedWallet(t, destinationUser.ID, 0)

	_, _, err := env.TransactionService.Transfer(context.Background(), dto.TransferRequest{
		SourceWalletID:      sourceWallet.ID,
		DestinationWalletID: destinationWallet.ID,
		AmountMinor:         2500,
	}, "insufficient-key")
	assertAppErrorCode(t, err, "insufficient_funds")
}

func TestTransactionServiceTransferSameWalletValidation(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "same-wallet@example.com")
	wallet := env.SeedWallet(t, user.ID, 10000)

	_, _, err := env.TransactionService.Transfer(context.Background(), dto.TransferRequest{
		SourceWalletID:      wallet.ID,
		DestinationWalletID: wallet.ID,
		AmountMinor:         100,
	}, "same-wallet-key")
	assertAppErrorCode(t, err, "validation_error")
}

func TestTransactionServiceTransferMissingWallet(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	sourceUser := env.SeedUser(t, "missing-source@example.com")
	sourceWallet := env.SeedWallet(t, sourceUser.ID, 10000)

	_, _, err := env.TransactionService.Transfer(context.Background(), dto.TransferRequest{
		SourceWalletID:      sourceWallet.ID,
		DestinationWalletID: uuid.New(),
		AmountMinor:         100,
	}, "missing-wallet-key")
	assertAppErrorCode(t, err, "resource_not_found")
}

func TestTransactionServiceTransferInactiveWallet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		sourceStatus      model.WalletStatus
		destinationStatus model.WalletStatus
	}{
		{
			name:              "inactive source wallet",
			sourceStatus:      model.WalletStatusSuspended,
			destinationStatus: model.WalletStatusActive,
		},
		{
			name:              "inactive destination wallet",
			sourceStatus:      model.WalletStatusActive,
			destinationStatus: model.WalletStatusSuspended,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := testutil.NewTestEnv(t)
			sourceUser := env.SeedUser(t, "inactive-source@example.com")
			destinationUser := env.SeedUser(t, "inactive-destination@example.com")
			sourceWallet := env.SeedWalletWithStatus(t, sourceUser.ID, 10000, tt.sourceStatus)
			destinationWallet := env.SeedWalletWithStatus(t, destinationUser.ID, 0, tt.destinationStatus)

			_, _, err := env.TransactionService.Transfer(context.Background(), dto.TransferRequest{
				SourceWalletID:      sourceWallet.ID,
				DestinationWalletID: destinationWallet.ID,
				AmountMinor:         500,
			}, "inactive-transfer-key")
			assertAppErrorCode(t, err, "wallet_not_active")
		})
	}
}

func TestTransactionServiceTransferIdempotencyReplay(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	sourceUser := env.SeedUser(t, "source-idem@example.com")
	destinationUser := env.SeedUser(t, "destination-idem@example.com")
	sourceWallet := env.SeedWallet(t, sourceUser.ID, 10000)
	destinationWallet := env.SeedWallet(t, destinationUser.ID, 0)

	request := dto.TransferRequest{
		SourceWalletID:      sourceWallet.ID,
		DestinationWalletID: destinationWallet.ID,
		AmountMinor:         2500,
	}

	first, firstStatus, err := env.TransactionService.Transfer(context.Background(), request, "idem-key")
	if err != nil {
		t.Fatalf("first transfer: %v", err)
	}
	second, secondStatus, err := env.TransactionService.Transfer(context.Background(), request, "idem-key")
	if err != nil {
		t.Fatalf("second transfer: %v", err)
	}

	if firstStatus != 201 || secondStatus != 201 {
		t.Fatalf("expected status 201 for both calls, got %d and %d", firstStatus, secondStatus)
	}
	if first.ID != second.ID {
		t.Fatalf("expected replayed transaction ID %s, got %s", first.ID, second.ID)
	}

	assertCount(t, env, &model.Transaction{}, "type = ?", model.TransactionTypeTransfer, 1)
}

func TestTransactionServiceTransferIdempotencyConflict(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	sourceUser := env.SeedUser(t, "source-transfer-conflict@example.com")
	destinationUser := env.SeedUser(t, "destination-transfer-conflict@example.com")
	sourceWallet := env.SeedWallet(t, sourceUser.ID, 10000)
	destinationWallet := env.SeedWallet(t, destinationUser.ID, 0)

	mustTransfer(t, env, sourceWallet.ID, destinationWallet.ID, 2500, "transfer-conflict-key")

	_, _, err := env.TransactionService.Transfer(context.Background(), dto.TransferRequest{
		SourceWalletID:      sourceWallet.ID,
		DestinationWalletID: destinationWallet.ID,
		AmountMinor:         2600,
	}, "transfer-conflict-key")
	assertAppErrorCode(t, err, "idempotency_conflict")
	assertCount(t, env, &model.Transaction{}, "type = ?", model.TransactionTypeTransfer, 1)
}

func TestTransactionServiceReverseRestoresBalances(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	sourceUser := env.SeedUser(t, "source-reverse@example.com")
	destinationUser := env.SeedUser(t, "destination-reverse@example.com")
	sourceWallet := env.SeedWallet(t, sourceUser.ID, 10000)
	destinationWallet := env.SeedWallet(t, destinationUser.ID, 0)

	transfer := mustTransfer(t, env, sourceWallet.ID, destinationWallet.ID, 2500, "reverse-transfer-key")

	reversal, statusCode, err := env.TransactionService.Reverse(context.Background(), transfer.ID, "reverse-key")
	if err != nil {
		t.Fatalf("reverse transaction: %v", err)
	}
	if statusCode != 201 {
		t.Fatalf("expected status 201, got %d", statusCode)
	}
	if reversal.RelatedTransactionID == nil || *reversal.RelatedTransactionID != transfer.ID {
		t.Fatalf("expected related transaction %s", transfer.ID)
	}

	assertWalletBalance(t, env, sourceWallet.ID, 10000)
	assertWalletBalance(t, env, destinationWallet.ID, 0)

	var original model.Transaction
	if err := env.DB.First(&original, "id = ?", transfer.ID).Error; err != nil {
		t.Fatalf("load original transaction: %v", err)
	}
	if original.Status != model.TransactionStatusReversed {
		t.Fatalf("expected original status reversed, got %s", original.Status)
	}
}

func TestTransactionServiceReverseMissingTransaction(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	_, _, err := env.TransactionService.Reverse(context.Background(), uuid.New(), "reverse-missing-key")
	assertAppErrorCode(t, err, "resource_not_found")
}

func TestTransactionServiceReverseAlreadyReversed(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	sourceUser := env.SeedUser(t, "source-already-reversed@example.com")
	destinationUser := env.SeedUser(t, "destination-already-reversed@example.com")
	sourceWallet := env.SeedWallet(t, sourceUser.ID, 10000)
	destinationWallet := env.SeedWallet(t, destinationUser.ID, 0)

	transfer := mustTransfer(t, env, sourceWallet.ID, destinationWallet.ID, 2500, "reverse-once-key")
	if _, _, err := env.TransactionService.Reverse(context.Background(), transfer.ID, "reverse-once"); err != nil {
		t.Fatalf("initial reverse: %v", err)
	}

	_, _, err := env.TransactionService.Reverse(context.Background(), transfer.ID, "reverse-twice")
	assertAppErrorCode(t, err, "already_reversed")
	assertCount(t, env, &model.Transaction{}, "type = ?", model.TransactionTypeReversal, 1)
}

func TestTransactionServiceReverseInsufficientFunds(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	sourceUser := env.SeedUser(t, "source-reverse-insufficient@example.com")
	destinationUser := env.SeedUser(t, "destination-reverse-insufficient@example.com")
	sourceWallet := env.SeedWallet(t, sourceUser.ID, 10000)
	destinationWallet := env.SeedWallet(t, destinationUser.ID, 0)

	transfer := mustTransfer(t, env, sourceWallet.ID, destinationWallet.ID, 2500, "reverse-insufficient-transfer")

	destination, err := env.WalletRepo.GetByID(context.Background(), destinationWallet.ID)
	if err != nil {
		t.Fatalf("load destination wallet: %v", err)
	}
	destination.BalanceCachedMinor = 1000
	if err := env.WalletRepo.Update(context.Background(), destination); err != nil {
		t.Fatalf("update destination wallet: %v", err)
	}

	_, _, err = env.TransactionService.Reverse(context.Background(), transfer.ID, "reverse-insufficient-key")
	assertAppErrorCode(t, err, "insufficient_funds")
	assertCount(t, env, &model.Transaction{}, "type = ?", model.TransactionTypeReversal, 0)

	var original model.Transaction
	if err := env.DB.First(&original, "id = ?", transfer.ID).Error; err != nil {
		t.Fatalf("load original transaction: %v", err)
	}
	if original.Status != model.TransactionStatusCompleted {
		t.Fatalf("expected original transaction to remain completed, got %s", original.Status)
	}
}

func TestTransactionServiceGetSuccess(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	user := env.SeedUser(t, "get-transaction@example.com")
	wallet := env.SeedWallet(t, user.ID, 0)
	transaction := mustTopUp(t, env, wallet.ID, 5000, "get-transaction-key")

	response, err := env.TransactionService.Get(context.Background(), transaction.ID)
	if err != nil {
		t.Fatalf("get transaction: %v", err)
	}

	if response.ID != transaction.ID {
		t.Fatalf("expected transaction ID %s, got %s", transaction.ID, response.ID)
	}
	if response.Type != model.TransactionTypeTopUp {
		t.Fatalf("expected topup type, got %s", response.Type)
	}
}

func TestTransactionServiceGetMissingTransaction(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	_, err := env.TransactionService.Get(context.Background(), uuid.New())
	assertAppErrorCode(t, err, "resource_not_found")
}

func TestTransactionServiceListRespectsFiltersAndPagination(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	sourceUser := env.SeedUser(t, "list-source@example.com")
	destinationUser := env.SeedUser(t, "list-destination@example.com")
	sourceWallet := env.SeedWallet(t, sourceUser.ID, 0)
	destinationWallet := env.SeedWallet(t, destinationUser.ID, 0)

	mustTopUp(t, env, sourceWallet.ID, 10000, "list-topup-key")
	transfer := mustTransfer(t, env, sourceWallet.ID, destinationWallet.ID, 2500, "list-transfer-key")
	filterType := model.TransactionTypeTransfer
	filterStatus := model.TransactionStatusCompleted

	response, err := env.TransactionService.List(context.Background(), 1, 1, repository.TransactionFilters{
		WalletID: &sourceWallet.ID,
		Type:     &filterType,
		Status:   &filterStatus,
	})
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}

	if response.Page != 1 || response.Limit != 1 {
		t.Fatalf("expected page 1 limit 1, got page %d limit %d", response.Page, response.Limit)
	}
	if response.Total != 1 {
		t.Fatalf("expected total 1, got %d", response.Total)
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(response.Items))
	}
	if response.Items[0].ID != transfer.ID {
		t.Fatalf("expected listed transfer ID %s, got %s", transfer.ID, response.Items[0].ID)
	}
}

func TestTransactionServiceConcurrentTransfersPreserveBalance(t *testing.T) {
	t.Parallel()

	env := testutil.NewTestEnv(t)
	sourceUser := env.SeedUser(t, "source-concurrency@example.com")
	destinationUser := env.SeedUser(t, "destination-concurrency@example.com")
	sourceWallet := env.SeedWallet(t, sourceUser.ID, 10000)
	destinationWallet := env.SeedWallet(t, destinationUser.ID, 0)

	request := dto.TransferRequest{
		SourceWalletID:      sourceWallet.ID,
		DestinationWalletID: destinationWallet.ID,
		AmountMinor:         7000,
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	var successCount int
	var mu sync.Mutex

	runTransfer := func(key string) {
		defer wg.Done()
		<-start

		_, _, err := env.TransactionService.Transfer(context.Background(), request, key)
		if err == nil {
			mu.Lock()
			successCount++
			mu.Unlock()
		}
	}

	wg.Add(2)
	go runTransfer("concurrency-key-1")
	go runTransfer("concurrency-key-2")
	close(start)
	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly one successful transfer, got %d", successCount)
	}

	var updatedSource model.Wallet
	if err := env.DB.First(&updatedSource, "id = ?", sourceWallet.ID).Error; err != nil {
		t.Fatalf("load source wallet: %v", err)
	}
	if updatedSource.BalanceCachedMinor < 0 {
		t.Fatalf("source balance must not be negative, got %d", updatedSource.BalanceCachedMinor)
	}
}

func mustTopUp(t *testing.T, env *testutil.TestEnv, walletID uuid.UUID, amount int64, key string) dto.TransactionResponse {
	t.Helper()

	response, statusCode, err := env.TransactionService.TopUp(context.Background(), dto.TopUpRequest{
		WalletID:    walletID,
		AmountMinor: amount,
	}, key)
	if err != nil {
		t.Fatalf("top up: %v", err)
	}
	if statusCode != 201 {
		t.Fatalf("expected topup status 201, got %d", statusCode)
	}

	return response
}

func mustTransfer(t *testing.T, env *testutil.TestEnv, sourceWalletID, destinationWalletID uuid.UUID, amount int64, key string) dto.TransactionResponse {
	t.Helper()

	response, statusCode, err := env.TransactionService.Transfer(context.Background(), dto.TransferRequest{
		SourceWalletID:      sourceWalletID,
		DestinationWalletID: destinationWalletID,
		AmountMinor:         amount,
	}, key)
	if err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if statusCode != 201 {
		t.Fatalf("expected transfer status 201, got %d", statusCode)
	}

	return response
}

func assertWalletBalance(t *testing.T, env *testutil.TestEnv, walletID uuid.UUID, want int64) {
	t.Helper()

	wallet, err := env.WalletRepo.GetByID(context.Background(), walletID)
	if err != nil {
		t.Fatalf("load wallet %s: %v", walletID, err)
	}
	if wallet.BalanceCachedMinor != want {
		t.Fatalf("expected wallet %s balance %d, got %d", walletID, want, wallet.BalanceCachedMinor)
	}
}

func assertCount(t *testing.T, env *testutil.TestEnv, modelValue any, query string, arg any, want int64) {
	t.Helper()

	var got int64
	if err := env.DB.Model(modelValue).Where(query, arg).Count(&got).Error; err != nil {
		t.Fatalf("count rows for %T: %v", modelValue, err)
	}
	if got != want {
		t.Fatalf("expected %d rows for %T, got %d", want, modelValue, got)
	}
}
