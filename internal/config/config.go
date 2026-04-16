package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultHTTPPort                = "8080"
	defaultDBDSN                   = "host=localhost user=testWallet password=Wallet11# dbname=wallet_service port=5432 sslmode=disable TimeZone=UTC"
	defaultRedisAddr               = "localhost:6379"
	defaultRedisDB                 = 0
	defaultBalanceCacheTTL         = 5 * time.Minute
	defaultOutboxPollInterval      = 2 * time.Second
	defaultOutboxBatchSize         = 20
	defaultIdempotencyTTL          = 24 * time.Hour
	defaultShutdownTimeout         = 10 * time.Second
	defaultPublisherClaimTimeout   = 30 * time.Second
	defaultTransactionPageLimit    = 20
	defaultCollectionMaxPageLimit  = 100
	defaultCollectionDefaultOffset = 20
)

// Config holds application configuration.
type Config struct {
	AppName               string
	HTTPPort              string
	DBDSN                 string
	RedisAddr             string
	RedisPassword         string
	RedisDB               int
	BalanceCacheTTL       time.Duration
	OutboxPollInterval    time.Duration
	OutboxBatchSize       int
	IdempotencyTTL        time.Duration
	ShutdownTimeout       time.Duration
	DefaultPageLimit      int
	MaxPageLimit          int
	PublisherClaimTimeout time.Duration
}

// Load reads configuration from the environment.
func Load() (Config, error) {
	cfg := Config{
		AppName:               envString("APP_NAME", "wallet-service-mvp"),
		HTTPPort:              envString("HTTP_PORT", defaultHTTPPort),
		DBDSN:                 envString("DB_DSN", defaultDBDSN),
		RedisAddr:             envString("REDIS_ADDR", defaultRedisAddr),
		RedisPassword:         envString("REDIS_PASSWORD", ""),
		RedisDB:               envInt("REDIS_DB", defaultRedisDB),
		BalanceCacheTTL:       envDuration("BALANCE_CACHE_TTL", defaultBalanceCacheTTL),
		OutboxPollInterval:    envDuration("OUTBOX_POLL_INTERVAL", defaultOutboxPollInterval),
		OutboxBatchSize:       envInt("OUTBOX_BATCH_SIZE", defaultOutboxBatchSize),
		IdempotencyTTL:        envDuration("IDEMPOTENCY_TTL", defaultIdempotencyTTL),
		ShutdownTimeout:       envDuration("SHUTDOWN_TIMEOUT", defaultShutdownTimeout),
		DefaultPageLimit:      envInt("DEFAULT_PAGE_LIMIT", defaultTransactionPageLimit),
		MaxPageLimit:          envInt("MAX_PAGE_LIMIT", defaultCollectionMaxPageLimit),
		PublisherClaimTimeout: envDuration("PUBLISHER_CLAIM_TIMEOUT", defaultPublisherClaimTimeout),
	}

	if cfg.OutboxBatchSize <= 0 {
		return Config{}, fmt.Errorf("OUTBOX_BATCH_SIZE must be positive")
	}
	if cfg.DefaultPageLimit <= 0 {
		return Config{}, fmt.Errorf("DEFAULT_PAGE_LIMIT must be positive")
	}
	if cfg.MaxPageLimit < cfg.DefaultPageLimit {
		return Config{}, fmt.Errorf("MAX_PAGE_LIMIT must be greater than or equal to DEFAULT_PAGE_LIMIT")
	}

	return cfg, nil
}

func envString(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}

func envInt(key string, fallback int) int {
	value := envString(key, "")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := envString(key, "")
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}
