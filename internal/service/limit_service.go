package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/digital-wallet/internal/config"
	"github.com/digital-wallet/internal/dao"
	"github.com/digital-wallet/internal/dbmodel"
	"github.com/google/uuid"
)

type LimitService struct {
	limitDAO dao.LimitStore
	cfg      *config.Config
}

func NewLimitService(limitDAO dao.LimitStore, cfg *config.Config) *LimitService {
	return &LimitService{
		limitDAO: limitDAO,
		cfg:      cfg,
	}
}

func (s *LimitService) CheckDailyLimit(ctx context.Context, walletID string, amount float64) error {
	limit, err := s.limitDAO.GetByWalletIDAndPeriod(ctx, walletID, "daily")
	if err != nil {
		return err
	}

	if limit == nil {
		// Create daily limit
		limit = &dbmodel.TransactionLimit{
			ID:       uuid.New().String(),
			WalletID: walletID,
			Period:   "daily",
			Amount:   "0.00",
			Limit:    fmt.Sprintf("%.2f", s.cfg.Limits.DailyLimit),
			ResetAt:  time.Now().Add(24 * time.Hour),
		}
		if err := s.limitDAO.Create(ctx, limit); err != nil {
			return err
		}
	}

	// Check if reset needed
	if time.Now().After(limit.ResetAt) {
		_ = s.limitDAO.ResetLimitIfNeeded(ctx, walletID, "daily", time.Now().Add(24*time.Hour))
		limit.Amount = "0.00"
	}

	currentAmount, _ := strconv.ParseFloat(limit.Amount, 64)
	limitAmount, _ := strconv.ParseFloat(limit.Limit, 64)

	if currentAmount+amount > limitAmount {
		return dbmodel.NewDomainError(
			dbmodel.ErrDailyLimitExceeded,
			"Daily transaction limit exceeded",
		).WithDetail("limit", limitAmount).WithDetail("current", currentAmount).WithDetail("requested", amount)
	}

	return nil
}

func (s *LimitService) CheckWeeklyLimit(ctx context.Context, walletID string, amount float64) error {
	limit, err := s.limitDAO.GetByWalletIDAndPeriod(ctx, walletID, "weekly")
	if err != nil {
		return err
	}

	if limit == nil {
		// Create weekly limit
		limit = &dbmodel.TransactionLimit{
			ID:       uuid.New().String(),
			WalletID: walletID,
			Period:   "weekly",
			Amount:   "0.00",
			Limit:    fmt.Sprintf("%.2f", s.cfg.Limits.WeeklyLimit),
			ResetAt:  time.Now().Add(7 * 24 * time.Hour),
		}
		if err := s.limitDAO.Create(ctx, limit); err != nil {
			return err
		}
	}

	// Check if reset needed
	if time.Now().After(limit.ResetAt) {
		_ = s.limitDAO.ResetLimitIfNeeded(ctx, walletID, "weekly", time.Now().Add(7*24*time.Hour))
		limit.Amount = "0.00"
	}

	currentAmount, _ := strconv.ParseFloat(limit.Amount, 64)
	limitAmount, _ := strconv.ParseFloat(limit.Limit, 64)

	if currentAmount+amount > limitAmount {
		return dbmodel.NewDomainError(
			dbmodel.ErrWeeklyLimitExceeded,
			"Weekly transaction limit exceeded",
		).WithDetail("limit", limitAmount).WithDetail("current", currentAmount).WithDetail("requested", amount)
	}

	return nil
}

func (s *LimitService) UpdateDailyLimit(ctx context.Context, walletID string, amount float64) error {
	limit, err := s.limitDAO.GetByWalletIDAndPeriod(ctx, walletID, "daily")
	if err != nil {
		return err
	}

	if limit == nil {
		return nil
	}

	return s.limitDAO.UpdateAmount(ctx, limit.ID, amount)
}

func (s *LimitService) UpdateWeeklyLimit(ctx context.Context, walletID string, amount float64) error {
	limit, err := s.limitDAO.GetByWalletIDAndPeriod(ctx, walletID, "weekly")
	if err != nil {
		return err
	}

	if limit == nil {
		return nil
	}

	return s.limitDAO.UpdateAmount(ctx, limit.ID, amount)
}
