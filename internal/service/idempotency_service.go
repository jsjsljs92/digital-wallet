package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/digital-wallet/internal/dbmodel"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IdempotencyService struct {
	db *gorm.DB
}

func NewIdempotencyService(db *gorm.DB) *IdempotencyService {
	return &IdempotencyService{db: db}
}

func (s *IdempotencyService) StoreResponse(ctx context.Context, walletID, idempotencyKey string, response interface{}, statusCode int) error {
	respBytes, err := json.Marshal(response)
	if err != nil {
		return err
	}

	record := &dbmodel.IdempotencyRecord{
		ID:             uuid.New().String(),
		WalletID:       walletID,
		IdempotencyKey: idempotencyKey,
		Response:       string(respBytes),
		StatusCode:     statusCode,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	return s.db.WithContext(ctx).Create(record).Error
}

func (s *IdempotencyService) GetResponse(ctx context.Context, walletID, idempotencyKey string) (string, int, error) {
	var record dbmodel.IdempotencyRecord
	if err := s.db.WithContext(ctx).
		Where("wallet_id = ? AND idempotency_key = ?", walletID, idempotencyKey).
		First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", 0, nil
		}
		return "", 0, err
	}

	if time.Now().After(record.ExpiresAt) {
		_ = s.db.WithContext(ctx).Delete(&record).Error
		return "", 0, nil
	}

	return record.Response, record.StatusCode, nil
}

func (s *IdempotencyService) CleanupExpired(ctx context.Context) error {
	return s.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&dbmodel.IdempotencyRecord{}).Error
}
