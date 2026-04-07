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
	"github.com/digital-wallet/internal/dbmanager"
	"github.com/digital-wallet/internal/routes"
	"github.com/go-chi/chi/v5"
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
	router := chi.NewRouter()
	routes.SetupRoutes(router, cfg, db, redis)

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
