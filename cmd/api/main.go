package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go-digital-wallet/internal/cache"
	"go-digital-wallet/internal/config"
	"go-digital-wallet/internal/event"
	"go-digital-wallet/internal/handler"
	"go-digital-wallet/internal/repository"
	"go-digital-wallet/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	db, err := repository.OpenPostgres(ctx, cfg.DBDSN)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}

	if err := repository.AutoMigrate(ctx, db); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}
	if err := repository.SeedSystemAccounts(ctx, db); err != nil {
		log.Fatalf("seed system accounts: %v", err)
	}

	redisClient := cache.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	cacheStore := cache.NewRedisBalanceCache(redisClient, cfg.BalanceCacheTTL)
	streamPublisher := event.NewRedisStreamPublisher(redisClient)

	userRepo := repository.NewUserRepository(db)
	walletRepo := repository.NewWalletRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	idempotencyRepo := repository.NewIdempotencyRepository(db)
	outboxRepo := repository.NewOutboxRepository(db)

	userService := service.NewUserService(userRepo, cfg.DefaultPageLimit, cfg.MaxPageLimit)
	walletService := service.NewWalletService(logger, userRepo, walletRepo, cacheStore)
	transactionService := service.NewTransactionService(
		logger,
		db,
		walletRepo,
		transactionRepo,
		idempotencyRepo,
		outboxRepo,
		cacheStore,
		cfg.IdempotencyTTL,
		cfg.DefaultPageLimit,
		cfg.MaxPageLimit,
	)

	outboxWorker := event.NewWorker(logger, outboxRepo, streamPublisher, cfg.OutboxBatchSize, event.StreamName)
	outboxWorker.Start(ctx, cfg.OutboxPollInterval)

	router := handler.NewRouter(logger, db, cacheStore, userService, walletService, transactionService)
	server := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: router,
	}

	go func() {
		logger.Info("starting api server", "port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server exited unexpectedly", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown server", "error", err)
	}

	if err := redisClient.Close(); err != nil {
		logger.Error("close redis client", "error", err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		if err := sqlDB.Close(); err != nil {
			logger.Error("close database", "error", err)
		}
	}
}
