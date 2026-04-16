package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go-digital-wallet/internal/apperror"
	"go-digital-wallet/internal/cache"
	"go-digital-wallet/internal/dto"
	"go-digital-wallet/internal/model"
	"go-digital-wallet/internal/repository"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type paginationConfig struct {
	defaultLimit int
	maxLimit     int
}

func normalizePage(page, limit int, cfg paginationConfig) (int, int) {
	return repository.NormalizePagination(page, limit, cfg.defaultLimit, cfg.maxLimit)
}

func hashRequest(payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request for idempotency hash: %w", err)
	}

	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func newReferenceCode(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.NewString())
}

func toJSONMap(metadata map[string]any) datatypes.JSONMap {
	if len(metadata) == 0 {
		return nil
	}

	return datatypes.JSONMap(metadata)
}

func replayResponse[T any](record *model.IdempotencyKey) (T, int, error) {
	var zero T
	if record == nil || record.ResponseStatusCode == nil || len(record.ResponseBodyJSON) == 0 {
		return zero, 0, nil
	}

	var response T
	if err := json.Unmarshal(record.ResponseBodyJSON, &response); err != nil {
		return zero, 0, fmt.Errorf("unmarshal idempotent response: %w", err)
	}

	return response, *record.ResponseStatusCode, nil
}

func touchedWallets(cacheStore cache.BalanceCache, logger *slog.Logger, ctx context.Context, wallets ...model.Wallet) {
	if cacheStore == nil {
		return
	}

	for _, wallet := range wallets {
		if err := cacheStore.Set(ctx, wallet.ID, wallet.BalanceCachedMinor); err != nil {
			logger.Warn("refresh balance cache", "wallet_id", wallet.ID, "error", err)
		}
	}
}

func getIdempotencyReplay[T any](
	ctx context.Context,
	repo *repository.IdempotencyRepository,
	key string,
	endpoint string,
	requestHash string,
) (T, int, *model.IdempotencyKey, error) {
	var zero T
	if key == "" {
		return zero, 0, nil, nil
	}

	record, err := repo.Get(ctx, key, endpoint)
	if err != nil {
		if repository.IsNotFound(err) {
			return zero, 0, nil, nil
		}
		return zero, 0, nil, apperror.Internal(err)
	}

	if record.RequestHash != requestHash {
		return zero, 0, nil, apperror.IdempotencyConflict()
	}

	response, statusCode, err := replayResponse[T](record)
	if err != nil {
		return zero, 0, nil, apperror.Internal(err)
	}

	return response, statusCode, record, nil
}

func createIdempotencyRecord(
	ctx context.Context,
	repo *repository.IdempotencyRepository,
	key string,
	endpoint string,
	requestHash string,
	ttl time.Duration,
) (*model.IdempotencyKey, error) {
	if key == "" {
		return nil, nil
	}

	record := &model.IdempotencyKey{
		Key:         key,
		Endpoint:    endpoint,
		RequestHash: requestHash,
		ExpiresAt:   time.Now().UTC().Add(ttl),
	}

	if err := repo.Create(ctx, record); err != nil {
		if repository.IsDuplicate(err) {
			return nil, apperror.Conflict("idempotency key is already in use")
		}
		return nil, apperror.Internal(err)
	}

	return record, nil
}

func withTransaction(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(fn)
}

func responseList[T any](items []T, page, limit int, total int64) dto.PageResponse[T] {
	return dto.PageResponse[T]{
		Items: items,
		Page:  page,
		Limit: limit,
		Total: total,
	}
}
