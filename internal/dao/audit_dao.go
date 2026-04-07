package dao

import (
	"context"
	"time"

	"github.com/digital-wallet/internal/dbmodel"
	"gorm.io/gorm"
)

type AuditDAO struct {
	db *gorm.DB
}

func NewAuditDAO(db *gorm.DB) *AuditDAO {
	return &AuditDAO{db: db}
}

func (d *AuditDAO) Create(ctx context.Context, log *dbmodel.AuditLog) error {
	return d.db.WithContext(ctx).Create(log).Error
}

func (d *AuditDAO) ListByWalletID(ctx context.Context, walletID string) ([]dbmodel.AuditLog, error) {
	var logs []dbmodel.AuditLog
	if err := d.db.WithContext(ctx).
		Where("wallet_id = ?", walletID).
		Order("created_at DESC").
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func (d *AuditDAO) GetSince(ctx context.Context, walletID string, since time.Time) ([]dbmodel.AuditLog, error) {
	var logs []dbmodel.AuditLog
	if err := d.db.WithContext(ctx).
		Where("wallet_id = ? AND created_at >= ?", walletID, since).
		Order("created_at DESC").
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
