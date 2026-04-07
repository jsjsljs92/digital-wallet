package dbmanager

import (
	"fmt"
	"log"
	"time"

	"github.com/digital-wallet/internal/config"
	"github.com/digital-wallet/internal/dbmodel"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type MySQLManager struct {
	db *gorm.DB
}

func NewMySQLManager(cfg *config.Config) (*MySQLManager, error) {
	dsn := cfg.Database.DSN()

	gormLogger := logger.Default.LogMode(logger.Info)
	if cfg.Server.Env == "production" {
		gormLogger = logger.Default.LogMode(logger.Error)
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.Database.MaxConnections)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxConnections / 2)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connected successfully")

	return &MySQLManager{db: db}, nil
}

func (m *MySQLManager) GetDB() *gorm.DB {
	return m.db
}

func (m *MySQLManager) Close() error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (m *MySQLManager) RunMigrations() error {
	return m.db.AutoMigrate(
		&dbmodel.User{},
		&dbmodel.Wallet{},
		&dbmodel.Transaction{},
		&dbmodel.TransactionLimit{},
		&dbmodel.AuditLog{},
		&dbmodel.IdempotencyRecord{},
	)
}

func (m *MySQLManager) Health() error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func (m *MySQLManager) CreateIndexes() error {
	if !m.db.Migrator().HasIndex("wallets", "user_id") {
		if err := m.db.Migrator().CreateIndex("wallets", "user_id"); err != nil {
			return err
		}
	}

	if !m.db.Migrator().HasIndex("transactions", "wallet_id") {
		if err := m.db.Migrator().CreateIndex("transactions", "wallet_id"); err != nil {
			return err
		}
	}

	if !m.db.Migrator().HasIndex("transactions", "idempotency_key") {
		if err := m.db.Migrator().CreateIndex("transactions", "idempotency_key"); err != nil {
			return err
		}
	}

	if !m.db.Migrator().HasIndex("audit_logs", "wallet_id") {
		if err := m.db.Migrator().CreateIndex("audit_logs", "wallet_id"); err != nil {
			return err
		}
	}

	return nil
}

func (m *MySQLManager) WithTx(txFunc func(*gorm.DB) error) error {
	tx := m.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if err := txFunc(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}
