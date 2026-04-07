package dao

import (
	"context"
	"errors"
	"time"

	"github.com/digital-wallet/internal/dbmodel"
	"gorm.io/gorm"
)

type LimitDAO struct {
	db *gorm.DB
}

func NewLimitDAO(db *gorm.DB) *LimitDAO {
	return &LimitDAO{db: db}
}

func (d *LimitDAO) Create(ctx context.Context, limit *dbmodel.TransactionLimit) error {
	return d.db.WithContext(ctx).Create(limit).Error
}

func (d *LimitDAO) GetByWalletIDAndPeriod(ctx context.Context, walletID, period string) (*dbmodel.TransactionLimit, error) {
	var limit dbmodel.TransactionLimit
	if err := d.db.WithContext(ctx).
		Where("wallet_id = ? AND period = ?", walletID, period).
		First(&limit).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &limit, nil
}

func (d *LimitDAO) Update(ctx context.Context, limit *dbmodel.TransactionLimit) error {
	return d.db.WithContext(ctx).Save(limit).Error
}

func (d *LimitDAO) UpdateAmount(ctx context.Context, limitID string, additionalAmount float64) error {
	// Note: In real implementation, would use DECIMAL addition
	return d.db.WithContext(ctx).
		Model(&dbmodel.TransactionLimit{}).
		Where("id = ?", limitID).
		Update("amount", gorm.Expr("amount + ?", additionalAmount)).Error
}

func (d *LimitDAO) ResetLimitIfNeeded(ctx context.Context, walletID, period string, resetTime time.Time) error {
	limit, err := d.GetByWalletIDAndPeriod(ctx, walletID, period)
	if err != nil {
		return err
	}

	if limit == nil || limit.ResetAt.Before(time.Now()) {
		// Limit needs reset or doesn't exist
		if limit == nil {
			return nil
		}

		return d.db.WithContext(ctx).
			Model(&dbmodel.TransactionLimit{}).
			Where("id = ?", limit.ID).
			Updates(map[string]interface{}{
				"amount":   0,
				"reset_at": resetTime,
			}).Error
	}

	return nil
}

func (d *LimitDAO) GetAllLimitsForWallet(ctx context.Context, walletID string) ([]dbmodel.TransactionLimit, error) {
	var limits []dbmodel.TransactionLimit
	if err := d.db.WithContext(ctx).
		Where("wallet_id = ?", walletID).
		Find(&limits).Error; err != nil {
		return nil, err
	}
	return limits, nil
}
