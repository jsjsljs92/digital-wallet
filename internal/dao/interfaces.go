package dao

import (
	"context"
	"time"

	"github.com/digital-wallet/internal/dbmodel"
)

// WalletStore defines the persistence operations required by services.
// Concrete implementations: *WalletDAO
type WalletStore interface {
	Create(ctx context.Context, wallet *dbmodel.Wallet) error
	GetByID(ctx context.Context, walletID string) (*dbmodel.Wallet, error)
	GetByUserID(ctx context.Context, userID string) (*dbmodel.Wallet, error)
	UpdateWithOptimisticLock(ctx context.Context, walletID string, updateFn func(*dbmodel.Wallet) error) error
}

type TransactionListFilters struct {
	Type      *string
	FromTime  *time.Time
	ToTime    *time.Time
	MinAmount *float64
	MaxAmount *float64
}

// TransactionStore defines the transaction persistence operations required by services.
// Concrete implementations: *TransactionDAO
type TransactionStore interface {
	Create(ctx context.Context, transaction *dbmodel.Transaction) error
	GetByIdempotencyKey(ctx context.Context, walletID, key string) (*dbmodel.Transaction, error)
	ListByWalletID(ctx context.Context, walletID string, limit, offset int) ([]dbmodel.Transaction, int64, error)
	ListByWalletIDFiltered(ctx context.Context, walletID string, filters TransactionListFilters, limit, offset int) ([]dbmodel.Transaction, int64, error)
	CountRecentTransactions(ctx context.Context, walletID string, windowMinutes int) (int64, error)
}

// LimitStore defines the transaction-limit persistence operations required by services.
// Concrete implementations: *LimitDAO
type LimitStore interface {
	Create(ctx context.Context, limit *dbmodel.TransactionLimit) error
	GetByWalletIDAndPeriod(ctx context.Context, walletID, period string) (*dbmodel.TransactionLimit, error)
	UpdateAmount(ctx context.Context, limitID string, additionalAmount float64) error
	ResetLimitIfNeeded(ctx context.Context, walletID, period string, resetTime time.Time) error
}

// AuditStore defines the audit-log persistence operations required by services.
// Concrete implementations: *AuditDAO
type AuditStore interface {
	Create(ctx context.Context, log *dbmodel.AuditLog) error
}
