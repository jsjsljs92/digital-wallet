package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/digital-wallet/internal/dbmodel"
	"gorm.io/gorm"
)

type TransactionDAO struct {
	db *gorm.DB
}

func NewTransactionDAO(db *gorm.DB) *TransactionDAO {
	return &TransactionDAO{db: db}
}

func (d *TransactionDAO) Create(ctx context.Context, transaction *dbmodel.Transaction) error {
	return d.db.WithContext(ctx).Create(transaction).Error
}

func (d *TransactionDAO) GetByID(ctx context.Context, transactionID string) (*dbmodel.Transaction, error) {
	var transaction dbmodel.Transaction
	if err := d.db.WithContext(ctx).Where("id = ?", transactionID).First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (d *TransactionDAO) GetByIdempotencyKey(ctx context.Context, walletID, key string) (*dbmodel.Transaction, error) {
	var transaction dbmodel.Transaction
	if err := d.db.WithContext(ctx).
		Where("wallet_id = ? AND idempotency_key = ?", walletID, key).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (d *TransactionDAO) ListByWalletID(ctx context.Context, walletID string, limit, offset int) ([]dbmodel.Transaction, int64, error) {
	var transactions []dbmodel.Transaction
	var total int64

	if err := d.db.WithContext(ctx).
		Model(&dbmodel.Transaction{}).
		Where("wallet_id = ?", walletID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := d.db.WithContext(ctx).
		Where("wallet_id = ?", walletID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

func (d *TransactionDAO) ListByWalletIDAndType(ctx context.Context, walletID, txType string, limit, offset int) ([]dbmodel.Transaction, int64, error) {
	var transactions []dbmodel.Transaction
	var total int64

	if err := d.db.WithContext(ctx).
		Model(&dbmodel.Transaction{}).
		Where("wallet_id = ? AND type = ?", walletID, txType).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := d.db.WithContext(ctx).
		Where("wallet_id = ? AND type = ?", walletID, txType).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

func (d *TransactionDAO) ListByWalletIDAndDateRange(ctx context.Context, walletID string, fromTime, toTime interface{}, limit, offset int) ([]dbmodel.Transaction, int64, error) {
	var transactions []dbmodel.Transaction
	var total int64

	query := d.db.WithContext(ctx).
		Model(&dbmodel.Transaction{}).
		Where("wallet_id = ? AND created_at BETWEEN ? AND ?", walletID, fromTime, toTime)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

func (d *TransactionDAO) CountRecentTransactions(ctx context.Context, walletID string, windowMinutes int) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&dbmodel.Transaction{}).
		Where("wallet_id = ? AND created_at >= DATE_SUB(NOW(), INTERVAL ? MINUTE)", walletID, windowMinutes).
		Count(&count).Error
	return count, err
}

func (d *TransactionDAO) ListByWalletIDFiltered(ctx context.Context, walletID string, filters TransactionListFilters, limit, offset int) ([]dbmodel.Transaction, int64, error) {
	var transactions []dbmodel.Transaction
	var total int64

	query := d.db.WithContext(ctx).
		Model(&dbmodel.Transaction{}).
		Where("wallet_id = ?", walletID)

	if filters.Type != nil && *filters.Type != "" {
		query = query.Where("type = ?", *filters.Type)
	}

	if filters.FromTime != nil {
		query = query.Where("created_at >= ?", *filters.FromTime)
	}

	if filters.ToTime != nil {
		query = query.Where("created_at <= ?", *filters.ToTime)
	}

	// `amount` is a DECIMAL column in MySQL; pass a normalized decimal string.
	if filters.MinAmount != nil {
		query = query.Where("amount >= ?", fmt.Sprintf("%.2f", *filters.MinAmount))
	}

	if filters.MaxAmount != nil {
		query = query.Where("amount <= ?", fmt.Sprintf("%.2f", *filters.MaxAmount))
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}
