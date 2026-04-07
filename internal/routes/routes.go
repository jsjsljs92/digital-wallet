package routes

import (
	"net/http"

	"github.com/digital-wallet/internal/config"
	"github.com/digital-wallet/internal/controller"
	"github.com/digital-wallet/internal/dao"
	"github.com/digital-wallet/internal/middleware"
	"github.com/digital-wallet/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// SetupRoutes initializes all routes for the API
func SetupRoutes(router chi.Router, cfg *config.Config, db *gorm.DB, redisClient *redis.Client) {
	// Initialize DAOs
	walletDAO := dao.NewWalletDAO(db)
	transactionDAO := dao.NewTransactionDAO(db)
	limitDAO := dao.NewLimitDAO(db)
	auditDAO := dao.NewAuditDAO(db)

	// Initialize Services
	fraudService := service.NewFraudService(transactionDAO, cfg)
	limitService := service.NewLimitService(limitDAO, cfg)
	transactionService := service.NewTransactionService(walletDAO, transactionDAO, limitDAO, auditDAO, fraudService, limitService)
	walletService := service.NewWalletService(walletDAO, auditDAO)

	// Initialize Controllers
	walletController := controller.NewWalletController(walletService)
	transactionController := controller.NewTransactionController(transactionService, walletService)

	// Apply global middleware
	router.Use(middleware.ErrorHandler)
	router.Use(middleware.Logger)
	router.Use(middleware.RequestID)
	router.Use(middleware.RateLimit(redisClient, cfg))
	router.Use(middleware.Auth(cfg))
	router.Use(corsMiddleware)

	// API v1 routes
	router.Route("/v1", func(r chi.Router) {
		// Health check
		r.Get("/health", healthHandler)

		// Wallet routes
		r.Post("/wallets", walletController.CreateWallet)
		r.Get("/wallets", walletController.GetWallet)

		// Transaction routes
		r.Post("/transactions/deposit", transactionController.Deposit)
		r.Post("/transactions/withdraw", transactionController.Withdraw)
		r.Get("/transactions", transactionController.GetTransactions)
	})
}

// healthHandler returns the health status of the service
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// corsMiddleware handles CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-User-ID, Idempotency-Key")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, X-RateLimit-Limit, X-RateLimit-Remaining")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
