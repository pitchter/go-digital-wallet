package service

import (
	"context"
	"log/slog"

	"go-digital-wallet/internal/apperror"
	"go-digital-wallet/internal/cache"
	"go-digital-wallet/internal/dto"
	"go-digital-wallet/internal/model"
	"go-digital-wallet/internal/repository"

	"github.com/google/uuid"
)

// WalletService handles wallet operations.
type WalletService struct {
	logger     *slog.Logger
	users      *repository.UserRepository
	wallets    *repository.WalletRepository
	cacheStore cache.BalanceCache
}

// NewWalletService constructs a wallet service.
func NewWalletService(
	logger *slog.Logger,
	users *repository.UserRepository,
	wallets *repository.WalletRepository,
	cacheStore cache.BalanceCache,
) *WalletService {
	return &WalletService{
		logger:     logger,
		users:      users,
		wallets:    wallets,
		cacheStore: cacheStore,
	}
}

// Create creates a wallet for a user.
func (s *WalletService) Create(ctx context.Context, request dto.CreateWalletRequest) (dto.WalletResponse, error) {
	user, err := s.users.GetByID(ctx, request.UserID)
	if err != nil {
		if repository.IsNotFound(err) {
			return dto.WalletResponse{}, apperror.NotFound("requested resource was not found")
		}
		return dto.WalletResponse{}, apperror.Internal(err)
	}

	if user.Status != model.UserStatusActive {
		return dto.WalletResponse{}, apperror.Validation("one or more fields are invalid")
	}

	wallet := model.Wallet{
		UserID:             request.UserID,
		Currency:           request.Currency,
		Status:             model.WalletStatusActive,
		BalanceCachedMinor: 0,
	}
	if wallet.Currency == "" {
		wallet.Currency = model.CurrencyTHB
	}

	if err := s.wallets.Create(ctx, &wallet); err != nil {
		if repository.IsDuplicate(err) {
			return dto.WalletResponse{}, apperror.Conflict("resource conflicts with existing data")
		}
		return dto.WalletResponse{}, apperror.Internal(err)
	}

	return dto.WalletFromModel(wallet), nil
}

// Get fetches a wallet.
func (s *WalletService) Get(ctx context.Context, id uuid.UUID) (dto.WalletResponse, error) {
	wallet, err := s.wallets.GetByID(ctx, id)
	if err != nil {
		if repository.IsNotFound(err) {
			return dto.WalletResponse{}, apperror.NotFound("requested resource was not found")
		}
		return dto.WalletResponse{}, apperror.Internal(err)
	}

	return dto.WalletFromModel(*wallet), nil
}

// GetBalance returns the current wallet balance with cache fallback.
func (s *WalletService) GetBalance(ctx context.Context, id uuid.UUID) (dto.BalanceResponse, error) {
	wallet, err := s.wallets.GetByID(ctx, id)
	if err != nil {
		if repository.IsNotFound(err) {
			return dto.BalanceResponse{}, apperror.NotFound("requested resource was not found")
		}
		return dto.BalanceResponse{}, apperror.Internal(err)
	}

	if s.cacheStore != nil {
		balance, found, cacheErr := s.cacheStore.Get(ctx, id)
		if cacheErr == nil && found {
			s.logger.Info("wallet balance served", "wallet_id", id, "cache_source", "redis")
			return dto.BalanceResponse{
				WalletID:     wallet.ID,
				Currency:     wallet.Currency,
				BalanceMinor: balance,
				Source:       "redis",
			}, nil
		}

		if cacheErr != nil {
			s.logger.Warn("wallet balance cache miss with error", "wallet_id", id, "error", cacheErr)
		}
	}

	if s.cacheStore != nil {
		if err := s.cacheStore.Set(ctx, wallet.ID, wallet.BalanceCachedMinor); err != nil {
			s.logger.Warn("prime balance cache", "wallet_id", wallet.ID, "error", err)
		}
	}

	s.logger.Info("wallet balance served", "wallet_id", id, "cache_source", "database")

	return dto.BalanceResponse{
		WalletID:     wallet.ID,
		Currency:     wallet.Currency,
		BalanceMinor: wallet.BalanceCachedMinor,
		Source:       "database",
	}, nil
}

// UpdateStatus updates wallet status.
func (s *WalletService) UpdateStatus(ctx context.Context, id uuid.UUID, status model.WalletStatus) (dto.WalletResponse, error) {
	wallet, err := s.wallets.GetByID(ctx, id)
	if err != nil {
		if repository.IsNotFound(err) {
			return dto.WalletResponse{}, apperror.NotFound("requested resource was not found")
		}
		return dto.WalletResponse{}, apperror.Internal(err)
	}

	wallet.Status = status
	if err := s.wallets.Update(ctx, wallet); err != nil {
		return dto.WalletResponse{}, apperror.Internal(err)
	}

	return dto.WalletFromModel(*wallet), nil
}
