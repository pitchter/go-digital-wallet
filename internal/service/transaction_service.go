package service

import (
	"context"
	"log/slog"
	"time"

	"go-digital-wallet/internal/apperror"
	"go-digital-wallet/internal/cache"
	"go-digital-wallet/internal/dto"
	"go-digital-wallet/internal/event"
	"go-digital-wallet/internal/model"
	"go-digital-wallet/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	topUpEndpoint    = "POST:/api/v1/transactions/topup"
	transferEndpoint = "POST:/api/v1/transactions/transfer"
	reverseEndpoint  = "POST:/api/v1/transactions/reverse"
)

// TransactionService handles money movement and transaction reads.
type TransactionService struct {
	logger          *slog.Logger
	db              *gorm.DB
	wallets         *repository.WalletRepository
	transactions    *repository.TransactionRepository
	idempotencyKeys *repository.IdempotencyRepository
	outboxEvents    *repository.OutboxRepository
	cacheStore      cache.BalanceCache
	idempotencyTTL  time.Duration
	pagination      paginationConfig
}

// NewTransactionService constructs a transaction service.
func NewTransactionService(
	logger *slog.Logger,
	db *gorm.DB,
	wallets *repository.WalletRepository,
	transactions *repository.TransactionRepository,
	idempotencyKeys *repository.IdempotencyRepository,
	outboxEvents *repository.OutboxRepository,
	cacheStore cache.BalanceCache,
	idempotencyTTL time.Duration,
	defaultLimit int,
	maxLimit int,
) *TransactionService {
	return &TransactionService{
		logger:          logger,
		db:              db,
		wallets:         wallets,
		transactions:    transactions,
		idempotencyKeys: idempotencyKeys,
		outboxEvents:    outboxEvents,
		cacheStore:      cacheStore,
		idempotencyTTL:  idempotencyTTL,
		pagination: paginationConfig{
			defaultLimit: defaultLimit,
			maxLimit:     maxLimit,
		},
	}
}

// TopUp posts a top-up transaction.
func (s *TransactionService) TopUp(ctx context.Context, request dto.TopUpRequest, idempotencyKey string) (dto.TransactionResponse, int, error) {
	requestHash, err := hashRequest(request)
	if err != nil {
		return dto.TransactionResponse{}, 0, apperror.Internal(err)
	}

	replayed, statusCode, _, err := getIdempotencyReplay[dto.TransactionResponse](ctx, s.idempotencyKeys, idempotencyKey, topUpEndpoint, requestHash)
	if err != nil {
		return dto.TransactionResponse{}, 0, err
	}
	if statusCode != 0 {
		return replayed, statusCode, nil
	}

	var response dto.TransactionResponse
	var touched []model.Wallet

	err = withTransaction(ctx, s.db, func(tx *gorm.DB) error {
		walletRepo := s.wallets.WithTx(tx)
		transactionRepo := s.transactions.WithTx(tx)
		idempotencyRepo := s.idempotencyKeys.WithTx(tx)
		outboxRepo := s.outboxEvents.WithTx(tx)

		record, err := createIdempotencyRecord(ctx, idempotencyRepo, idempotencyKey, topUpEndpoint, requestHash, s.idempotencyTTL)
		if err != nil {
			return err
		}

		wallets, err := walletRepo.GetByIDsForUpdate(ctx, model.SystemWalletID, request.WalletID)
		if err != nil {
			return apperror.Internal(err)
		}

		systemWallet, destinationWallet, lookupErr := resolveTopUpWallets(wallets, request.WalletID)
		if lookupErr != nil {
			return lookupErr
		}
		if !destinationWallet.IsActive() {
			return apperror.WalletNotActive()
		}

		transaction := model.Transaction{
			ReferenceCode:       newReferenceCode("TOPUP"),
			Type:                model.TransactionTypeTopUp,
			Status:              model.TransactionStatusCompleted,
			SourceWalletID:      &systemWallet.ID,
			DestinationWalletID: &destinationWallet.ID,
			AmountMinor:         request.AmountMinor,
			Currency:            model.CurrencyTHB,
			IdempotencyKey:      optionalString(idempotencyKey),
			MetadataJSON:        toJSONMap(request.Metadata),
		}
		if err := transactionRepo.Create(ctx, &transaction); err != nil {
			return apperror.Internal(err)
		}

		systemWallet.BalanceCachedMinor -= request.AmountMinor
		destinationWallet.BalanceCachedMinor += request.AmountMinor

		if err := walletRepo.Update(ctx, systemWallet); err != nil {
			return apperror.Internal(err)
		}
		if err := walletRepo.Update(ctx, destinationWallet); err != nil {
			return apperror.Internal(err)
		}

		entries := []model.LedgerEntry{
			{
				TransactionID: transaction.ID,
				WalletID:      systemWallet.ID,
				EntryType:     model.LedgerEntryTypeDebit,
				AmountMinor:   request.AmountMinor,
			},
			{
				TransactionID: transaction.ID,
				WalletID:      destinationWallet.ID,
				EntryType:     model.LedgerEntryTypeCredit,
				AmountMinor:   request.AmountMinor,
			},
		}
		if err := transactionRepo.CreateLedgerEntries(ctx, entries); err != nil {
			return apperror.Internal(err)
		}

		outboxEvent, err := event.NewOutboxEvent("transaction.completed", transaction)
		if err != nil {
			return apperror.Internal(err)
		}
		if err := outboxRepo.Create(ctx, outboxEvent); err != nil {
			return apperror.Internal(err)
		}

		response = dto.TransactionFromModel(transaction)
		touched = []model.Wallet{*systemWallet, *destinationWallet}

		if record != nil {
			if err := idempotencyRepo.SaveResponse(ctx, record.ID, 201, response, &transaction.ID); err != nil {
				return apperror.Internal(err)
			}
		}

		return nil
	})
	if err != nil {
		return dto.TransactionResponse{}, 0, err
	}

	touchedWallets(s.cacheStore, s.logger, ctx, touched...)

	return response, 201, nil
}

// Transfer posts a transfer transaction.
func (s *TransactionService) Transfer(ctx context.Context, request dto.TransferRequest, idempotencyKey string) (dto.TransactionResponse, int, error) {
	if request.SourceWalletID == request.DestinationWalletID {
		return dto.TransactionResponse{}, 0, apperror.Validation("one or more fields are invalid")
	}

	requestHash, err := hashRequest(request)
	if err != nil {
		return dto.TransactionResponse{}, 0, apperror.Internal(err)
	}

	replayed, statusCode, _, err := getIdempotencyReplay[dto.TransactionResponse](ctx, s.idempotencyKeys, idempotencyKey, transferEndpoint, requestHash)
	if err != nil {
		return dto.TransactionResponse{}, 0, err
	}
	if statusCode != 0 {
		return replayed, statusCode, nil
	}

	var response dto.TransactionResponse
	var touched []model.Wallet

	err = withTransaction(ctx, s.db, func(tx *gorm.DB) error {
		walletRepo := s.wallets.WithTx(tx)
		transactionRepo := s.transactions.WithTx(tx)
		idempotencyRepo := s.idempotencyKeys.WithTx(tx)
		outboxRepo := s.outboxEvents.WithTx(tx)

		record, err := createIdempotencyRecord(ctx, idempotencyRepo, idempotencyKey, transferEndpoint, requestHash, s.idempotencyTTL)
		if err != nil {
			return err
		}

		wallets, err := walletRepo.GetByIDsForUpdate(ctx, request.SourceWalletID, request.DestinationWalletID)
		if err != nil {
			return apperror.Internal(err)
		}

		sourceWallet, destinationWallet, lookupErr := resolveTransferWallets(wallets, request.SourceWalletID, request.DestinationWalletID)
		if lookupErr != nil {
			return lookupErr
		}
		if !sourceWallet.IsActive() || !destinationWallet.IsActive() {
			return apperror.WalletNotActive()
		}
		if sourceWallet.BalanceCachedMinor < request.AmountMinor {
			return apperror.InsufficientFunds()
		}

		transaction := model.Transaction{
			ReferenceCode:       newReferenceCode("TRF"),
			Type:                model.TransactionTypeTransfer,
			Status:              model.TransactionStatusCompleted,
			SourceWalletID:      &sourceWallet.ID,
			DestinationWalletID: &destinationWallet.ID,
			AmountMinor:         request.AmountMinor,
			Currency:            model.CurrencyTHB,
			IdempotencyKey:      optionalString(idempotencyKey),
			MetadataJSON:        toJSONMap(request.Metadata),
		}
		if err := transactionRepo.Create(ctx, &transaction); err != nil {
			return apperror.Internal(err)
		}

		sourceWallet.BalanceCachedMinor -= request.AmountMinor
		destinationWallet.BalanceCachedMinor += request.AmountMinor

		if err := walletRepo.Update(ctx, sourceWallet); err != nil {
			return apperror.Internal(err)
		}
		if err := walletRepo.Update(ctx, destinationWallet); err != nil {
			return apperror.Internal(err)
		}

		entries := []model.LedgerEntry{
			{
				TransactionID: transaction.ID,
				WalletID:      sourceWallet.ID,
				EntryType:     model.LedgerEntryTypeDebit,
				AmountMinor:   request.AmountMinor,
			},
			{
				TransactionID: transaction.ID,
				WalletID:      destinationWallet.ID,
				EntryType:     model.LedgerEntryTypeCredit,
				AmountMinor:   request.AmountMinor,
			},
		}
		if err := transactionRepo.CreateLedgerEntries(ctx, entries); err != nil {
			return apperror.Internal(err)
		}

		outboxEvent, err := event.NewOutboxEvent("transaction.completed", transaction)
		if err != nil {
			return apperror.Internal(err)
		}
		if err := outboxRepo.Create(ctx, outboxEvent); err != nil {
			return apperror.Internal(err)
		}

		response = dto.TransactionFromModel(transaction)
		touched = []model.Wallet{*sourceWallet, *destinationWallet}

		if record != nil {
			if err := idempotencyRepo.SaveResponse(ctx, record.ID, 201, response, &transaction.ID); err != nil {
				return apperror.Internal(err)
			}
		}

		return nil
	})
	if err != nil {
		return dto.TransactionResponse{}, 0, err
	}

	touchedWallets(s.cacheStore, s.logger, ctx, touched...)

	return response, 201, nil
}

// Reverse creates a compensating transaction.
func (s *TransactionService) Reverse(ctx context.Context, transactionID uuid.UUID, idempotencyKey string) (dto.TransactionResponse, int, error) {
	requestPayload := struct {
		TransactionID uuid.UUID `json:"transaction_id"`
	}{
		TransactionID: transactionID,
	}

	requestHash, err := hashRequest(requestPayload)
	if err != nil {
		return dto.TransactionResponse{}, 0, apperror.Internal(err)
	}

	replayed, statusCode, _, err := getIdempotencyReplay[dto.TransactionResponse](ctx, s.idempotencyKeys, idempotencyKey, reverseEndpoint, requestHash)
	if err != nil {
		return dto.TransactionResponse{}, 0, err
	}
	if statusCode != 0 {
		return replayed, statusCode, nil
	}

	var response dto.TransactionResponse
	var touched []model.Wallet

	err = withTransaction(ctx, s.db, func(tx *gorm.DB) error {
		walletRepo := s.wallets.WithTx(tx)
		transactionRepo := s.transactions.WithTx(tx)
		idempotencyRepo := s.idempotencyKeys.WithTx(tx)
		outboxRepo := s.outboxEvents.WithTx(tx)

		record, err := createIdempotencyRecord(ctx, idempotencyRepo, idempotencyKey, reverseEndpoint, requestHash, s.idempotencyTTL)
		if err != nil {
			return err
		}

		original, err := transactionRepo.GetByIDForUpdate(ctx, transactionID)
		if err != nil {
			if repository.IsNotFound(err) {
				return apperror.NotFound("requested resource was not found")
			}
			return apperror.Internal(err)
		}
		if original.Status == model.TransactionStatusReversed {
			return apperror.AlreadyReversed()
		}
		if original.Type != model.TransactionTypeTopUp && original.Type != model.TransactionTypeTransfer {
			return apperror.Validation("one or more fields are invalid")
		}
		if original.SourceWalletID == nil || original.DestinationWalletID == nil {
			return apperror.Validation("one or more fields are invalid")
		}

		wallets, err := walletRepo.GetByIDsForUpdate(ctx, *original.SourceWalletID, *original.DestinationWalletID)
		if err != nil {
			return apperror.Internal(err)
		}

		sourceWallet, destinationWallet, lookupErr := resolveTransferWallets(wallets, *original.SourceWalletID, *original.DestinationWalletID)
		if lookupErr != nil {
			return lookupErr
		}
		if destinationWallet.BalanceCachedMinor < original.AmountMinor {
			return apperror.InsufficientFunds()
		}

		reversal := model.Transaction{
			ReferenceCode:        newReferenceCode("REV"),
			Type:                 model.TransactionTypeReversal,
			Status:               model.TransactionStatusCompleted,
			SourceWalletID:       &destinationWallet.ID,
			DestinationWalletID:  &sourceWallet.ID,
			AmountMinor:          original.AmountMinor,
			Currency:             original.Currency,
			IdempotencyKey:       optionalString(idempotencyKey),
			RelatedTransactionID: &original.ID,
			MetadataJSON:         original.MetadataJSON,
		}
		if err := transactionRepo.Create(ctx, &reversal); err != nil {
			return apperror.Internal(err)
		}

		destinationWallet.BalanceCachedMinor -= original.AmountMinor
		sourceWallet.BalanceCachedMinor += original.AmountMinor

		if err := walletRepo.Update(ctx, sourceWallet); err != nil {
			return apperror.Internal(err)
		}
		if err := walletRepo.Update(ctx, destinationWallet); err != nil {
			return apperror.Internal(err)
		}

		entries := []model.LedgerEntry{
			{
				TransactionID: reversal.ID,
				WalletID:      destinationWallet.ID,
				EntryType:     model.LedgerEntryTypeDebit,
				AmountMinor:   original.AmountMinor,
			},
			{
				TransactionID: reversal.ID,
				WalletID:      sourceWallet.ID,
				EntryType:     model.LedgerEntryTypeCredit,
				AmountMinor:   original.AmountMinor,
			},
		}
		if err := transactionRepo.CreateLedgerEntries(ctx, entries); err != nil {
			return apperror.Internal(err)
		}

		original.Status = model.TransactionStatusReversed
		if err := transactionRepo.Update(ctx, original); err != nil {
			return apperror.Internal(err)
		}

		outboxEvent, err := event.NewOutboxEvent("transaction.reversed", reversal)
		if err != nil {
			return apperror.Internal(err)
		}
		if err := outboxRepo.Create(ctx, outboxEvent); err != nil {
			return apperror.Internal(err)
		}

		response = dto.TransactionFromModel(reversal)
		touched = []model.Wallet{*sourceWallet, *destinationWallet}

		if record != nil {
			if err := idempotencyRepo.SaveResponse(ctx, record.ID, 201, response, &reversal.ID); err != nil {
				return apperror.Internal(err)
			}
		}

		return nil
	})
	if err != nil {
		return dto.TransactionResponse{}, 0, err
	}

	touchedWallets(s.cacheStore, s.logger, ctx, touched...)

	return response, 201, nil
}

// Get fetches a single transaction.
func (s *TransactionService) Get(ctx context.Context, id uuid.UUID) (dto.TransactionResponse, error) {
	txModel, err := s.transactions.GetByID(ctx, id)
	if err != nil {
		if repository.IsNotFound(err) {
			return dto.TransactionResponse{}, apperror.NotFound("requested resource was not found")
		}
		return dto.TransactionResponse{}, apperror.Internal(err)
	}

	return dto.TransactionFromModel(*txModel), nil
}

// List returns paginated transactions.
func (s *TransactionService) List(
	ctx context.Context,
	page int,
	limit int,
	filters repository.TransactionFilters,
) (dto.PageResponse[dto.TransactionResponse], error) {
	page, limit = normalizePage(page, limit, s.pagination)

	txModels, total, err := s.transactions.List(ctx, page, limit, filters)
	if err != nil {
		return dto.PageResponse[dto.TransactionResponse]{}, apperror.Internal(err)
	}

	items := make([]dto.TransactionResponse, 0, len(txModels))
	for _, txModel := range txModels {
		items = append(items, dto.TransactionFromModel(txModel))
	}

	return responseList(items, page, limit, total), nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func resolveTopUpWallets(wallets []model.Wallet, destinationID uuid.UUID) (*model.Wallet, *model.Wallet, error) {
	var systemWallet *model.Wallet
	var destinationWallet *model.Wallet

	for i := range wallets {
		wallet := &wallets[i]
		switch wallet.ID {
		case model.SystemWalletID:
			systemWallet = wallet
		case destinationID:
			destinationWallet = wallet
		}
	}

	if systemWallet == nil || destinationWallet == nil {
		return nil, nil, apperror.NotFound("requested resource was not found")
	}

	return systemWallet, destinationWallet, nil
}

func resolveTransferWallets(wallets []model.Wallet, sourceID, destinationID uuid.UUID) (*model.Wallet, *model.Wallet, error) {
	var sourceWallet *model.Wallet
	var destinationWallet *model.Wallet

	for i := range wallets {
		wallet := &wallets[i]
		switch wallet.ID {
		case sourceID:
			sourceWallet = wallet
		case destinationID:
			destinationWallet = wallet
		}
	}

	if sourceWallet == nil || destinationWallet == nil {
		return nil, nil, apperror.NotFound("requested resource was not found")
	}

	return sourceWallet, destinationWallet, nil
}
