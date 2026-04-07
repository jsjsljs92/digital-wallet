package main

import (
	"context"
	"log"

	"github.com/digital-wallet/internal/config"
	"github.com/digital-wallet/internal/dao"
	"github.com/digital-wallet/internal/dbmanager"
	"github.com/digital-wallet/internal/dbmodel"
	"github.com/google/uuid"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dbMgr, err := dbmanager.NewMySQLManager(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbMgr.Close()

	// Run migrations first
	if err := dbMgr.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	db := dbMgr.GetDB()
	ctx := context.Background()

	// Create sample users and wallets
	walletDAO := dao.NewWalletDAO(db)

	for i := 1; i <= 5; i++ {
		walletID := "wallet_" + uuid.New().String()[:8]
		userID := "user_" + uuid.New().String()[:8]

		wallet := &dbmodel.Wallet{
			ID:      walletID,
			UserID:  userID,
			Balance: "1000.00",
			Status:  "active",
			Version: 0,
		}

		if err := walletDAO.Create(ctx, wallet); err != nil {
			log.Printf("Failed to create wallet: %v", err)
		} else {
			log.Printf("Created wallet %s for user %s with balance 1000.00", walletID, userID)
		}
	}

	log.Println("Seed data created successfully")
}
