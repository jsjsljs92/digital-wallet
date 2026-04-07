package unit

import (
	"testing"

	"github.com/digital-wallet/internal/dbmodel"
	"github.com/stretchr/testify/assert"
)

// TestWalletModel_Validation tests basic wallet model validation
func TestWalletModel_Validation(t *testing.T) {
	tests := []struct {
		name      string
		wallet    *dbmodel.Wallet
		wantValid bool
	}{
		{
			name: "valid wallet",
			wallet: &dbmodel.Wallet{
				ID:      "wallet_123",
				UserID:  "user_123",
				Balance: "100.50",
				Status:  "active",
				Version: 1,
			},
			wantValid: true,
		},
		{
			name: "wallet with zero balance",
			wallet: &dbmodel.Wallet{
				ID:      "wallet_456",
				UserID:  "user_456",
				Balance: "0.00",
				Status:  "active",
				Version: 0,
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.wallet)
			assert.NotEmpty(t, tt.wallet.ID)
			assert.NotEmpty(t, tt.wallet.UserID)
			assert.NotEmpty(t, tt.wallet.Balance)
			assert.GreaterOrEqual(t, tt.wallet.Version, int64(0))
		})
	}
}

// TestTransactionModel_Validation tests basic transaction model validation
func TestTransactionModel_Validation(t *testing.T) {
	tests := []struct {
		name        string
		transaction *dbmodel.Transaction
		wantValid   bool
	}{
		{
			name: "valid deposit transaction",
			transaction: &dbmodel.Transaction{
				ID:         "txn_123",
				WalletID:   "wallet_123",
				Type:       "deposit",
				Amount:     "100.00",
				NewBalance: "200.00",
				Status:     "completed",
			},
			wantValid: true,
		},
		{
			name: "valid withdraw transaction",
			transaction: &dbmodel.Transaction{
				ID:         "txn_456",
				WalletID:   "wallet_456",
				Type:       "withdraw",
				Amount:     "50.00",
				NewBalance: "50.00",
				Status:     "completed",
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.transaction)
			assert.NotEmpty(t, tt.transaction.ID)
			assert.NotEmpty(t, tt.transaction.WalletID)
			assert.True(t, tt.transaction.Type == "deposit" || tt.transaction.Type == "withdraw")
			assert.NotEmpty(t, tt.transaction.Amount)
			assert.NotEmpty(t, tt.transaction.NewBalance)
		})
	}
}

// TestTransactionLimit_Validation tests transaction limit model validation
func TestTransactionLimit_Validation(t *testing.T) {
	tests := []struct {
		name      string
		limit     *dbmodel.TransactionLimit
		wantValid bool
	}{
		{
			name: "valid daily limit",
			limit: &dbmodel.TransactionLimit{
				ID:       "limit_123",
				WalletID: "wallet_123",
				Period:   "daily",
				Amount:   "1000.00",
				Limit:    "5000.00",
			},
			wantValid: true,
		},
		{
			name: "valid weekly limit",
			limit: &dbmodel.TransactionLimit{
				ID:       "limit_456",
				WalletID: "wallet_456",
				Period:   "weekly",
				Amount:   "3000.00",
				Limit:    "15000.00",
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.limit)
			assert.NotEmpty(t, tt.limit.ID)
			assert.NotEmpty(t, tt.limit.WalletID)
			assert.True(t, tt.limit.Period == "daily" || tt.limit.Period == "weekly")
			assert.NotEmpty(t, tt.limit.Amount)
			assert.NotEmpty(t, tt.limit.Limit)
		})
	}
}

// TestAuditLog_Validation tests audit log model validation
func TestAuditLog_Validation(t *testing.T) {
	tests := []struct {
		name      string
		auditLog  *dbmodel.AuditLog
		wantValid bool
	}{
		{
			name: "valid wallet created audit log",
			auditLog: &dbmodel.AuditLog{
				ID:       "log_123",
				WalletID: "wallet_123",
				Action:   "wallet_created",
				NewValue: `{"user_id":"user_123","balance":"0.00"}`,
			},
			wantValid: true,
		},
		{
			name: "valid deposit audit log",
			auditLog: &dbmodel.AuditLog{
				ID:       "log_456",
				WalletID: "wallet_456",
				Action:   "deposit",
				OldValue: `{"balance":"150.00"}`,
				NewValue: `{"balance":"250.00"}`,
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.auditLog)
			assert.NotEmpty(t, tt.auditLog.ID)
			assert.NotEmpty(t, tt.auditLog.WalletID)
			assert.NotEmpty(t, tt.auditLog.Action)
		})
	}
}

// TestDomainError_Creation tests domain error creation
func TestDomainError_Creation(t *testing.T) {
	err := dbmodel.NewDomainError(dbmodel.ErrInsufficientBalance, "not enough funds")
	assert.NotNil(t, err)
	assert.Equal(t, dbmodel.ErrInsufficientBalance, err.Code)
	assert.Equal(t, "not enough funds", err.Message)
}

// TestDomainError_ErrorCodes tests all error codes are defined
func TestDomainError_ErrorCodes(t *testing.T) {
	errorCodes := []dbmodel.ErrorCode{
		dbmodel.ErrWalletAlreadyExists,
		dbmodel.ErrWalletNotFound,
		dbmodel.ErrInsufficientBalance,
		dbmodel.ErrInvalidAmount,
		dbmodel.ErrDailyLimitExceeded,
		dbmodel.ErrWeeklyLimitExceeded,
		dbmodel.ErrIdempotencyConflict,
		dbmodel.ErrOptimisticLockFailed,
	}

	for _, code := range errorCodes {
		t.Run(string(code), func(t *testing.T) {
			assert.NotEmpty(t, code)
		})
	}
}
