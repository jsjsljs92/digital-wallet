package main

import (
	"github.com/digital-wallet/internal/config"
	"github.com/digital-wallet/internal/controller"
	"github.com/digital-wallet/internal/dao"
	"github.com/digital-wallet/internal/dbmanager"
	"github.com/digital-wallet/internal/service"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Providers

func provideConfig() (*config.Config, error) {
	return config.Load()
}

func provideMySQL(cfg *config.Config) (*gorm.DB, error) {
	mgr, err := dbmanager.NewMySQLManager(cfg)
	if err != nil {
		return nil, err
	}
	return mgr.GetDB(), nil
}

func provideRedis(cfg *config.Config) (*redis.Client, error) {
	mgr, err := dbmanager.NewRedisManager(cfg)
	if err != nil {
		return nil, err
	}
	return mgr.GetClient(), nil
}

// DAOs

func provideWalletDAO(db *gorm.DB) *dao.WalletDAO {
	return dao.NewWalletDAO(db)
}

func provideTransactionDAO(db *gorm.DB) *dao.TransactionDAO {
	return dao.NewTransactionDAO(db)
}

func provideLimitDAO(db *gorm.DB) *dao.LimitDAO {
	return dao.NewLimitDAO(db)
}

func provideAuditDAO(db *gorm.DB) *dao.AuditDAO {
	return dao.NewAuditDAO(db)
}

// Services

func provideFraudService(transactionDAO *dao.TransactionDAO, cfg *config.Config) *service.FraudService {
	return service.NewFraudService(transactionDAO, cfg)
}

func provideLimitService(limitDAO *dao.LimitDAO, cfg *config.Config) *service.LimitService {
	return service.NewLimitService(limitDAO, cfg)
}

func provideTransactionService(
	walletDAO *dao.WalletDAO,
	transactionDAO *dao.TransactionDAO,
	auditDAO *dao.AuditDAO,
	fraudService *service.FraudService,
	limitService *service.LimitService,
) *service.TransactionService {
	return service.NewTransactionService(walletDAO, transactionDAO, auditDAO, fraudService, limitService)
}

func provideWalletService(walletDAO *dao.WalletDAO, auditDAO *dao.AuditDAO) *service.WalletService {
	return service.NewWalletService(walletDAO, auditDAO)
}

func provideIdempotencyService(db *gorm.DB) *service.IdempotencyService {
	return service.NewIdempotencyService(db)
}

// Controllers

func provideWalletController(walletService *service.WalletService) *controller.WalletController {
	return controller.NewWalletController(walletService)
}

func provideTransactionController(
	transactionService *service.TransactionService,
	walletService *service.WalletService,
) *controller.TransactionController {
	return controller.NewTransactionController(transactionService, walletService)
}

// Application

type Application struct {
	Config                *config.Config
	DB                    *gorm.DB
	Redis                 *redis.Client
	WalletController      *controller.WalletController
	TransactionController *controller.TransactionController
}

func provideApplication(
	cfg *config.Config,
	db *gorm.DB,
	redis *redis.Client,
	walletCtrl *controller.WalletController,
	txnCtrl *controller.TransactionController,
) *Application {
	return &Application{
		Config:                cfg,
		DB:                    db,
		Redis:                 redis,
		WalletController:      walletCtrl,
		TransactionController: txnCtrl,
	}
}

// Wire Set

var (
	configSet = wire.NewSet(provideConfig)

	databaseSet = wire.NewSet(
		provideMySQL,
		provideRedis,
	)

	daoSet = wire.NewSet(
		provideWalletDAO,
		provideTransactionDAO,
		provideLimitDAO,
		provideAuditDAO,
	)

	serviceSet = wire.NewSet(
		provideFraudService,
		provideLimitService,
		provideTransactionService,
		provideWalletService,
		provideIdempotencyService,
	)

	controllerSet = wire.NewSet(
		provideWalletController,
		provideTransactionController,
	)

	applicationSet = wire.NewSet(
		configSet,
		databaseSet,
		daoSet,
		serviceSet,
		controllerSet,
		provideApplication,
	)
)

func InitializeApplication() (*Application, error) {
	wire.Build(applicationSet)
	return nil, nil
}
