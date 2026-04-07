package main

import (
	"log"

	"github.com/digital-wallet/internal/config"
	"github.com/digital-wallet/internal/dbmanager"
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

	log.Println("Running migrations...")
	if err := dbMgr.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Creating indexes...")
	if err := dbMgr.CreateIndexes(); err != nil {
		log.Fatalf("Failed to create indexes: %v", err)
	}

	log.Println("Migrations completed successfully")
}
