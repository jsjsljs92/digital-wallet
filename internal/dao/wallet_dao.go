package dao

import (
	"context"
	"errors"

	"github.com/digital-wallet/internal/dbmodel"
	"gorm.io/gorm"
)

type WalletDAO struct {
	db *gorm.DB
}

func NewWalletDAO(db *gorm.DB) *WalletDAO {
	return &WalletDAO{db: db}
}

func (d *WalletDAO) Create(ctx context.Context, wallet *dbmodel.Wallet) error {
	return d.db.WithContext(ctx).Create(wallet).Error
}

func (d *WalletDAO) GetByID(ctx context.Context, walletID string) (*dbmodel.Wallet, error) {
	var wallet dbmodel.Wallet
	if err := d.db.WithContext(ctx).Where("id = ?", walletID).First(&wallet).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &wallet, nil
}

func (d *WalletDAO) GetByUserID(ctx context.Context, userID string) (*dbmodel.Wallet, error) {
	var wallet dbmodel.Wallet
	if err := d.db.WithContext(ctx).Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &wallet, nil
}

func (d *WalletDAO) Update(ctx context.Context, wallet *dbmodel.Wallet) error {
	return d.db.WithContext(ctx).Save(wallet).Error
}

func (d *WalletDAO) UpdateWithOptimisticLock(ctx context.Context, walletID string, updateFn func(*dbmodel.Wallet) error) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var wallet dbmodel.Wallet
		if err := tx.Where("id = ?", walletID).First(&wallet).Error; err != nil {
			return err
		}

		if err := updateFn(&wallet); err != nil {
			return err
		}

		currentVersion := wallet.Version
		wallet.Version++

		result := tx.Model(&dbmodel.Wallet{}).
			Where("id = ? AND version = ?", walletID, currentVersion).
			Updates(wallet)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return dbmodel.NewDomainError(
				dbmodel.ErrOptimisticLockFailed,
				"Wallet version mismatch, concurrent update detected",
			)
		}

		return nil
	})
}

func (d *WalletDAO) GetBalance(ctx context.Context, walletID string) (float64, error) {
	var wallet dbmodel.Wallet
	if err := d.db.WithContext(ctx).Select("balance").Where("id = ?", walletID).First(&wallet).Error; err != nil {
		return 0, err
	}
	// Note: In real implementation, parse the DECIMAL from database
	return 0, nil
}

func (d *WalletDAO) ListAll(ctx context.Context) ([]dbmodel.Wallet, error) {
	var wallets []dbmodel.Wallet
	if err := d.db.WithContext(ctx).Find(&wallets).Error; err != nil {
		return nil, err
	}
	return wallets, nil
}
