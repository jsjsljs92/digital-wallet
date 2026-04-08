package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/digital-wallet/internal/dao"
	"github.com/digital-wallet/internal/dbmodel"
	"github.com/google/uuid"
)

type TransactionService struct {
	walletDAO      *dao.WalletDAO
	transactionDAO *dao.TransactionDAO
	limitDAO       *dao.LimitDAO
	auditDAO       *dao.AuditDAO
	fraudService   *FraudService
	limitService   *LimitService
}

func NewTransactionService(
	walletDAO *dao.WalletDAO,
	transactionDAO *dao.TransactionDAO,
	limitDAO *dao.LimitDAO,
	auditDAO *dao.AuditDAO,
	fraudService *FraudService,
	limitService *LimitService,
) *TransactionService {
	return &TransactionService{
		walletDAO:      walletDAO,
		transactionDAO: transactionDAO,
		limitDAO:       limitDAO,
		auditDAO:       auditDAO,
		fraudService:   fraudService,
		limitService:   limitService,
	}
}

type DepositRequest struct {
	Amount         float64
	IdempotencyKey string
	Reason         string
}

type WithdrawRequest struct {
	Amount         float64
	IdempotencyKey string
	OTP            string
}

type TransactionResponse struct {
	TransactionID string  `json:"transaction_id"`
	WalletID      string  `json:"wallet_id"`
	Type          string  `json:"type"`
	Amount        float64 `json:"amount"`
	NewBalance    float64 `json:"new_balance"`
	Status        string  `json:"status"`
	FraudDetected bool    `json:"fraud_detected"`
	CreatedAt     string  `json:"created_at"`
}

const (
	maxRetries         = 3
	initialRetryWaitMs = 10
	retryBackoffFactor = 2.0
)

func (s *TransactionService) Deposit(ctx context.Context, walletID string, req *DepositRequest) (*TransactionResponse, error) {
	// Validate input
	if req.Amount <= 0 {
		return nil, dbmodel.NewDomainError(
			dbmodel.ErrInvalidAmount,
			"Amount must be greater than 0",
		).WithDetail("amount", req.Amount)
	}

	// Check idempotency
	if req.IdempotencyKey != "" {
		existing, err := s.transactionDAO.GetByIdempotencyKey(ctx, walletID, req.IdempotencyKey)
		if err != nil {
			return nil, err
		}

		if existing != nil {
			return s.transactionToResponse(existing), nil
		}
	}

	// Check daily limit
	if err := s.limitService.CheckDailyLimit(ctx, walletID, req.Amount); err != nil {
		return nil, err
	}

	// Check weekly limit
	if err := s.limitService.CheckWeeklyLimit(ctx, walletID, req.Amount); err != nil {
		return nil, err
	}

	// Perform optimistic locking retry
	var txResp *TransactionResponse
	var txErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		txResp, txErr = s.performDeposit(ctx, walletID, req)
		if txErr == nil {
			return txResp, nil
		}

		// Check if it's an optimistic lock failure
		if domErr, ok := txErr.(*dbmodel.DomainError); ok && domErr.Code == dbmodel.ErrOptimisticLockFailed {
			// Retry with backoff
			waitTime := time.Duration(initialRetryWaitMs*int(retryBackoffFactor)) * time.Millisecond
			time.Sleep(waitTime)
			continue
		}

		// Other error, return immediately
		return nil, txErr
	}

	return txResp, txErr
}

func (s *TransactionService) performDeposit(ctx context.Context, walletID string, req *DepositRequest) (*TransactionResponse, error) {
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

	currentBalance, _ := strconv.ParseFloat(wallet.Balance, 64)
	newBalance := currentBalance + req.Amount

	// Check fraud
	fraudDetected := s.fraudService.DetectFraud(ctx, walletID, req.Amount)

	// Update wallet with optimistic lock
	err = s.walletDAO.UpdateWithOptimisticLock(ctx, walletID, func(w *dbmodel.Wallet) error {
		w.Balance = fmt.Sprintf("%.2f", newBalance)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Create transaction record
	txID := "txn_" + uuid.New().String()[:8]
	transaction := &dbmodel.Transaction{
		ID:             txID,
		WalletID:       walletID,
		Type:           "deposit",
		Amount:         fmt.Sprintf("%.2f", req.Amount),
		NewBalance:     fmt.Sprintf("%.2f", newBalance),
		Status:         "completed",
		IdempotencyKey: req.IdempotencyKey,
		FraudDetected:  fraudDetected,
		Reason:         req.Reason,
		Metadata:       "{}",
	}

	if err := s.transactionDAO.Create(ctx, transaction); err != nil {
		return nil, err
	}

	// Update limits
	if err := s.limitService.UpdateDailyLimit(ctx, walletID, req.Amount); err != nil {
		return nil, err
	}

	if err := s.limitService.UpdateWeeklyLimit(ctx, walletID, req.Amount); err != nil {
		return nil, err
	}

	// Create audit log
	_ = s.auditDAO.Create(ctx, &dbmodel.AuditLog{
		ID:       uuid.New().String(),
		WalletID: walletID,
		Action:   "deposit",
		OldValue: fmt.Sprintf(`{"balance": %f}`, currentBalance),
		NewValue: fmt.Sprintf(`{"balance": %f}`, newBalance),
		Details:  fmt.Sprintf(`{"transaction_id": "%s", "amount": %f}`, txID, req.Amount),
	})

	return s.transactionToResponse(transaction), nil
}

func (s *TransactionService) Withdraw(ctx context.Context, walletID string, req *WithdrawRequest) (*TransactionResponse, error) {
	// Validate input
	if req.Amount <= 0 {
		return nil, dbmodel.NewDomainError(
			dbmodel.ErrInvalidAmount,
			"Amount must be greater than 0",
		).WithDetail("amount", req.Amount)
	}

	// Validate OTP (mock implementation)
	if req.OTP != "123456" {
		return nil, dbmodel.NewDomainError(
			dbmodel.ErrInvalidOTP,
			"Invalid OTP",
		)
	}

	// Check idempotency
	if req.IdempotencyKey != "" {
		existing, err := s.transactionDAO.GetByIdempotencyKey(ctx, walletID, req.IdempotencyKey)
		if err != nil {
			return nil, err
		}

		if existing != nil {
			return s.transactionToResponse(existing), nil
		}
	}

	// Perform optimistic locking retry
	var txResp *TransactionResponse
	var txErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		txResp, txErr = s.performWithdraw(ctx, walletID, req)
		if txErr == nil {
			return txResp, nil
		}

		// Check if it's an optimistic lock failure
		if domErr, ok := txErr.(*dbmodel.DomainError); ok && domErr.Code == dbmodel.ErrOptimisticLockFailed {
			time.Sleep(time.Duration(initialRetryWaitMs*int(retryBackoffFactor)) * time.Millisecond)
			continue
		}

		return nil, txErr
	}

	return txResp, txErr
}

func (s *TransactionService) performWithdraw(ctx context.Context, walletID string, req *WithdrawRequest) (*TransactionResponse, error) {
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

	currentBalance, _ := strconv.ParseFloat(wallet.Balance, 64)
	if currentBalance < req.Amount {
		return nil, dbmodel.NewDomainError(
			dbmodel.ErrInsufficientBalance,
			"Wallet balance is insufficient for this withdrawal",
		).WithDetail("required", req.Amount).WithDetail("available", currentBalance)
	}

	newBalance := currentBalance - req.Amount

	// Check fraud
	fraudDetected := s.fraudService.DetectFraud(ctx, walletID, req.Amount)

	// Update wallet with optimistic lock
	err = s.walletDAO.UpdateWithOptimisticLock(ctx, walletID, func(w *dbmodel.Wallet) error {
		w.Balance = fmt.Sprintf("%.2f", newBalance)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Create transaction record
	txID := "txn_" + uuid.New().String()[:8]
	transaction := &dbmodel.Transaction{
		ID:             txID,
		WalletID:       walletID,
		Type:           "withdraw",
		Amount:         fmt.Sprintf("%.2f", req.Amount),
		NewBalance:     fmt.Sprintf("%.2f", newBalance),
		Status:         "completed",
		IdempotencyKey: req.IdempotencyKey,
		FraudDetected:  fraudDetected,
		Metadata:       "{}",
	}

	if err := s.transactionDAO.Create(ctx, transaction); err != nil {
		return nil, err
	}

	// Update limits
	if err := s.limitService.UpdateDailyLimit(ctx, walletID, req.Amount); err != nil {
		return nil, err
	}

	if err := s.limitService.UpdateWeeklyLimit(ctx, walletID, req.Amount); err != nil {
		return nil, err
	}

	// Create audit log
	_ = s.auditDAO.Create(ctx, &dbmodel.AuditLog{
		ID:       uuid.New().String(),
		WalletID: walletID,
		Action:   "withdraw",
		OldValue: fmt.Sprintf(`{"balance": %f}`, currentBalance),
		NewValue: fmt.Sprintf(`{"balance": %f}`, newBalance),
		Details:  fmt.Sprintf(`{"transaction_id": "%s", "amount": %f}`, txID, req.Amount),
	})

	return s.transactionToResponse(transaction), nil
}

func (s *TransactionService) GetTransactionHistory(ctx context.Context, walletID string, limit, offset int) ([]TransactionResponse, int64, error) {
	transactions, total, err := s.transactionDAO.ListByWalletID(ctx, walletID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]TransactionResponse, len(transactions))
	for i, tx := range transactions {
		responses[i] = *s.transactionToResponse(&tx)
	}

	return responses, total, nil
}

func (s *TransactionService) transactionToResponse(tx *dbmodel.Transaction) *TransactionResponse {
	amount, _ := strconv.ParseFloat(tx.Amount, 64)
	newBalance, _ := strconv.ParseFloat(tx.NewBalance, 64)

	return &TransactionResponse{
		TransactionID: tx.ID,
		WalletID:      tx.WalletID,
		Type:          tx.Type,
		Amount:        amount,
		NewBalance:    newBalance,
		Status:        tx.Status,
		FraudDetected: tx.FraudDetected,
		CreatedAt:     tx.CreatedAt.String(),
	}
}
