package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/digital-wallet/internal/config"
	"github.com/digital-wallet/internal/controller"
	"github.com/digital-wallet/internal/dao"
	"github.com/digital-wallet/internal/dbmanager"
	"github.com/digital-wallet/internal/middleware"
	"github.com/digital-wallet/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	dbMgr, err := dbmanager.NewMySQLManager(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbMgr.Close()

	// Run migrations
	if err := dbMgr.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	if err := dbMgr.CreateIndexes(); err != nil {
		log.Fatalf("Failed to create indexes: %v", err)
	}

	// Initialize Redis
	redisMgr, err := dbmanager.NewRedisManager(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisMgr.Close()

	db := dbMgr.GetDB()
	redis := redisMgr.GetClient()

	// Setup HTTP server
	router := setupRouter(cfg, db, redis)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("Server starting on port %d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Server shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

func setupRouter(cfg *config.Config, db *gorm.DB, redis *redis.Client) chi.Router {
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

	// Setup router
	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.ErrorHandler)
	router.Use(middleware.Logger)
	router.Use(middleware.RequestID)
	router.Use(middleware.RateLimit(redis, cfg))
	router.Use(middleware.Auth(cfg))
	router.Use(corsMiddleware)

	// Routes
	router.Route("/v1", func(r chi.Router) {
		// Health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		})

		// Wallet routes
		walletController.RegisterRoutes(r)

		// Transaction routes
		transactionController.RegisterRoutes(r)
	})

	return router
}

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
