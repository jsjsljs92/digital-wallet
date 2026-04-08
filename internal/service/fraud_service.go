package service

import (
	"context"

	"github.com/digital-wallet/internal/config"
	"github.com/digital-wallet/internal/dao"
)

type FraudService struct {
	transactionDAO dao.TransactionStore
	cfg            *config.Config
}

func NewFraudService(transactionDAO dao.TransactionStore, cfg *config.Config) *FraudService {
	return &FraudService{
		transactionDAO: transactionDAO,
		cfg:            cfg,
	}
}

func (s *FraudService) DetectFraud(ctx context.Context, walletID string, amount float64) bool {
	// Check amount threshold
	if amount > s.cfg.Fraud.AmountThreshold {
		return true
	}

	// Check velocity
	count, err := s.transactionDAO.CountRecentTransactions(ctx, walletID, s.cfg.Fraud.VelocityWindowMins)
	if err != nil {
		return false
	}

	if count > int64(s.cfg.Fraud.VelocityThreshold) {
		return true
	}

	return false
}
