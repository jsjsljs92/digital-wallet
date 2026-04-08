package unit

import (
	"context"
	"testing"
	"time"

	"github.com/digital-wallet/internal/dao"
	"github.com/digital-wallet/internal/dbmodel"
	"github.com/digital-wallet/internal/service"
	"github.com/stretchr/testify/assert"
)

type mockWalletStore struct {
	getByIDCalls int
}

func (m *mockWalletStore) Create(ctx context.Context, wallet *dbmodel.Wallet) error { return nil }
func (m *mockWalletStore) GetByID(ctx context.Context, walletID string) (*dbmodel.Wallet, error) {
	m.getByIDCalls++
	return &dbmodel.Wallet{ID: walletID, UserID: "u1", Balance: "100.00", Version: 0, Status: "active"}, nil
}
func (m *mockWalletStore) GetByUserID(ctx context.Context, userID string) (*dbmodel.Wallet, error) {
	return &dbmodel.Wallet{ID: "wallet_1", UserID: userID, Balance: "0.00", Version: 0, Status: "active"}, nil
}
func (m *mockWalletStore) UpdateWithOptimisticLock(ctx context.Context, walletID string, updateFn func(*dbmodel.Wallet) error) error {
	return nil
}

type mockTransactionStore struct {
	listFilteredCalls int
	lastWalletID      string
	lastFilters       dao.TransactionListFilters
}

func (m *mockTransactionStore) Create(ctx context.Context, transaction *dbmodel.Transaction) error {
	return nil
}
func (m *mockTransactionStore) GetByIdempotencyKey(ctx context.Context, walletID, key string) (*dbmodel.Transaction, error) {
	return nil, nil
}
func (m *mockTransactionStore) ListByWalletID(ctx context.Context, walletID string, limit, offset int) ([]dbmodel.Transaction, int64, error) {
	return []dbmodel.Transaction{}, 0, nil
}
func (m *mockTransactionStore) ListByWalletIDFiltered(ctx context.Context, walletID string, filters dao.TransactionListFilters, limit, offset int) ([]dbmodel.Transaction, int64, error) {
	m.listFilteredCalls++
	m.lastWalletID = walletID
	m.lastFilters = filters
	return []dbmodel.Transaction{}, 0, nil
}
func (m *mockTransactionStore) CountRecentTransactions(ctx context.Context, walletID string, windowMinutes int) (int64, error) {
	return 0, nil
}

type mockAuditStore struct{}

func (m *mockAuditStore) Create(ctx context.Context, log *dbmodel.AuditLog) error { return nil }

type mockFraudDetector struct{}

func (m *mockFraudDetector) DetectFraud(ctx context.Context, walletID string, amount float64) bool {
	return false
}

type mockLimitEnforcer struct {
	dailyErr  error
	weeklyErr error

	dailyCalls  int
	weeklyCalls int
}

func (m *mockLimitEnforcer) CheckDailyLimit(ctx context.Context, walletID string, amount float64) error {
	m.dailyCalls++
	return m.dailyErr
}
func (m *mockLimitEnforcer) CheckWeeklyLimit(ctx context.Context, walletID string, amount float64) error {
	m.weeklyCalls++
	return m.weeklyErr
}
func (m *mockLimitEnforcer) UpdateDailyLimit(ctx context.Context, walletID string, amount float64) error {
	return nil
}
func (m *mockLimitEnforcer) UpdateWeeklyLimit(ctx context.Context, walletID string, amount float64) error {
	return nil
}

func TestTransactionService_Withdraw_EnforcesLimits(t *testing.T) {
	ctx := context.Background()

	walletStore := &mockWalletStore{}
	txStore := &mockTransactionStore{}
	auditStore := &mockAuditStore{}
	fraud := &mockFraudDetector{}

	limitErr := dbmodel.NewDomainError(dbmodel.ErrDailyLimitExceeded, "Daily transaction limit exceeded")
	limits := &mockLimitEnforcer{dailyErr: limitErr}

	svc := service.NewTransactionService(walletStore, txStore, auditStore, fraud, limits)

	_, err := svc.Withdraw(ctx, "wallet_1", &service.WithdrawRequest{
		Amount:         10,
		IdempotencyKey: "k1",
		OTP:            "123456",
	})

	assert.Error(t, err)
	// Ensure the specific limit error is returned.
	if domErr, ok := err.(*dbmodel.DomainError); ok {
		assert.Equal(t, dbmodel.ErrDailyLimitExceeded, domErr.Code)
	} else {
		t.Fatalf("expected DomainError, got %T", err)
	}

	// Limits should be checked; the wallet read should not be required when limit check fails fast.
	assert.Equal(t, 1, limits.dailyCalls)
	assert.Equal(t, 0, limits.weeklyCalls)
	assert.Equal(t, 0, walletStore.getByIDCalls)
}

func TestTransactionService_GetTransactionHistoryFiltered_PassesFilters(t *testing.T) {
	ctx := context.Background()

	walletStore := &mockWalletStore{}
	txStore := &mockTransactionStore{}
	auditStore := &mockAuditStore{}
	fraud := &mockFraudDetector{}
	limits := &mockLimitEnforcer{}

	svc := service.NewTransactionService(walletStore, txStore, auditStore, fraud, limits)

	txType := "deposit"
	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 8, 23, 59, 59, 0, time.UTC)
	minAmt := 25.0
	maxAmt := 100.0

	_, _, err := svc.GetTransactionHistoryFiltered(ctx, "wallet_abc", service.TransactionHistoryFilters{
		Type:      &txType,
		FromTime:  &from,
		ToTime:    &to,
		MinAmount: &minAmt,
		MaxAmount: &maxAmt,
	}, 10, 0)

	assert.NoError(t, err)
	assert.Equal(t, 1, txStore.listFilteredCalls)
	assert.Equal(t, "wallet_abc", txStore.lastWalletID)
	assert.NotNil(t, txStore.lastFilters.Type)
	assert.Equal(t, "deposit", *txStore.lastFilters.Type)
	assert.Equal(t, &from, txStore.lastFilters.FromTime)
	assert.Equal(t, &to, txStore.lastFilters.ToTime)
	assert.Equal(t, &minAmt, txStore.lastFilters.MinAmount)
	assert.Equal(t, &maxAmt, txStore.lastFilters.MaxAmount)
}
