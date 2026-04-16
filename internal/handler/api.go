package handler

import (
	"context"
	"log/slog"
	"net/http"

	"go-digital-wallet/internal/cache"
	"go-digital-wallet/internal/middleware"
	"go-digital-wallet/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// API wires HTTP handlers to services.
type API struct {
	logger       *slog.Logger
	db           *gorm.DB
	cacheStore   cache.BalanceCache
	users        *service.UserService
	wallets      *service.WalletService
	transactions *service.TransactionService
}

// NewRouter constructs the Gin router.
func NewRouter(
	logger *slog.Logger,
	db *gorm.DB,
	cacheStore cache.BalanceCache,
	users *service.UserService,
	wallets *service.WalletService,
	transactions *service.TransactionService,
) *gin.Engine {
	api := &API{
		logger:       logger,
		db:           db,
		cacheStore:   cacheStore,
		users:        users,
		wallets:      wallets,
		transactions: transactions,
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.RequestLogger(logger))

	router.GET("/healthz", api.healthz)
	router.GET("/readyz", api.readyz)

	v1 := router.Group("/api/v1")
	{
		v1.POST("/users", api.createUser)
		v1.GET("/users", api.listUsers)
		v1.GET("/users/:id", api.getUser)
		v1.PUT("/users/:id", api.updateUser)
		v1.DELETE("/users/:id", api.deleteUser)

		v1.POST("/wallets", api.createWallet)
		v1.GET("/wallets/:id", api.getWallet)
		v1.GET("/wallets/:id/balance", api.getWalletBalance)
		v1.PATCH("/wallets/:id/status", api.updateWalletStatus)

		v1.POST("/transactions/topup", api.topUp)
		v1.POST("/transactions/transfer", api.transfer)
		v1.GET("/transactions", api.listTransactions)
		v1.GET("/transactions/:id", api.getTransaction)
		v1.POST("/transactions/:id/reverse", api.reverseTransaction)
	}

	return router
}

func (a *API) healthz(c *gin.Context) {
	writeJSON(c, http.StatusOK, gin.H{"status": "ok"})
}

func (a *API) readyz(c *gin.Context) {
	status := "ready"
	dependencies := gin.H{}

	sqlDB, err := a.db.DB()
	if err != nil {
		writeError(c, err)
		return
	}
	if err := sqlDB.PingContext(c.Request.Context()); err != nil {
		writeError(c, err)
		return
	}
	dependencies["database"] = "up"

	if a.cacheStore != nil {
		if err := a.cacheStore.Ping(c.Request.Context()); err != nil {
			dependencies["redis"] = "down"
			a.logger.Warn("redis unavailable during readiness check", "error", err)
		} else {
			dependencies["redis"] = "up"
		}
	}

	writeJSON(c, http.StatusOK, gin.H{
		"status":       status,
		"dependencies": dependencies,
	})
}

func backgroundContext(c *gin.Context) context.Context {
	return c.Request.Context()
}
