package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/digital-wallet/internal/dao"
	"github.com/digital-wallet/internal/dbmodel"
	"github.com/google/uuid"
)

type WalletService struct {
	walletDAO dao.WalletStore
	auditDAO  dao.AuditStore
}

func NewWalletService(walletDAO dao.WalletStore, auditDAO dao.AuditStore) *WalletService {
	return &WalletService{
		walletDAO: walletDAO,
		auditDAO:  auditDAO,
	}
}

type CreateWalletRequest struct {
	UserID string
}

type WalletResponse struct {
	WalletID  string  `json:"wallet_id"`
	UserID    string  `json:"user_id"`
	Balance   float64 `json:"balance"`
	Status    string  `json:"status"`
	Version   int64   `json:"version"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func (s *WalletService) CreateWallet(ctx context.Context, req *CreateWalletRequest) (*WalletResponse, error) {
	// Check if wallet already exists
	existing, err := s.walletDAO.GetByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		return nil, dbmodel.NewDomainError(
			dbmodel.ErrWalletAlreadyExists,
			"Wallet already exists for this user",
		).WithDetail("user_id", req.UserID)
	}

	walletID := "wallet_" + uuid.New().String()[:8]

	wallet := &dbmodel.Wallet{
		ID:      walletID,
		UserID:  req.UserID,
		Balance: "0.00",
		Status:  "active",
		Version: 0,
	}

	if err := s.walletDAO.Create(ctx, wallet); err != nil {
		return nil, err
	}

	// Create audit log
	_ = s.auditDAO.Create(ctx, &dbmodel.AuditLog{
		ID:       uuid.New().String(),
		WalletID: walletID,
		Action:   "wallet_created",
		NewValue: fmt.Sprintf(`{"wallet_id": "%s", "user_id": "%s"}`, walletID, req.UserID),
	})

	return &WalletResponse{
		WalletID:  wallet.ID,
		UserID:    wallet.UserID,
		Balance:   0,
		Status:    wallet.Status,
		Version:   wallet.Version,
		CreatedAt: wallet.CreatedAt.String(),
		UpdatedAt: wallet.UpdatedAt.String(),
	}, nil
}

func (s *WalletService) GetWallet(ctx context.Context, walletID string) (*WalletResponse, error) {
	wallet, err := s.walletDAO.GetByID(ctx, walletID)
	if err != nil {
		return nil, err
	}

	if wallet == nil {
		return nil, dbmodel.NewDomainError(
			dbmodel.ErrWalletNotFound,
			"Wallet not found",
		).WithDetail("wallet_id", walletID)
	}

	balance, _ := strconv.ParseFloat(wallet.Balance, 64)

	return &WalletResponse{
		WalletID:  wallet.ID,
		UserID:    wallet.UserID,
		Balance:   balance,
		Status:    wallet.Status,
		Version:   wallet.Version,
		CreatedAt: wallet.CreatedAt.String(),
		UpdatedAt: wallet.UpdatedAt.String(),
	}, nil
}

func (s *WalletService) GetWalletByUserID(ctx context.Context, userID string) (*WalletResponse, error) {
	wallet, err := s.walletDAO.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if wallet == nil {
		return nil, dbmodel.NewDomainError(
			dbmodel.ErrWalletNotFound,
			"Wallet not found for user",
		).WithDetail("user_id", userID)
	}

	balance, _ := strconv.ParseFloat(wallet.Balance, 64)

	return &WalletResponse{
		WalletID:  wallet.ID,
		UserID:    wallet.UserID,
		Balance:   balance,
		Status:    wallet.Status,
		Version:   wallet.Version,
		CreatedAt: wallet.CreatedAt.String(),
		UpdatedAt: wallet.UpdatedAt.String(),
	}, nil
}
