package unit

import (
	"context"
	"testing"

	"github.com/digital-wallet/internal/config"
	"github.com/digital-wallet/internal/dao"
	"github.com/digital-wallet/internal/dbmodel"
	"github.com/digital-wallet/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockWalletDAO struct {
	mock.Mock
}

func (m *MockWalletDAO) Create(ctx context.Context, wallet *dbmodel.Wallet) error {
	args := m.Called(ctx, wallet)
	return args.Error(0)
}

func (m *MockWalletDAO) GetByID(ctx context.Context, walletID string) (*dbmodel.Wallet, error) {
	args := m.Called(ctx, walletID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dbmodel.Wallet), args.Error(1)
}

func (m *MockWalletDAO) GetByUserID(ctx context.Context, userID string) (*dbmodel.Wallet, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dbmodel.Wallet), args.Error(1)
}

func (m *MockWalletDAO) Update(ctx context.Context, wallet *dbmodel.Wallet) error {
	args := m.Called(ctx, wallet)
	return args.Error(0)
}

func (m *MockWalletDAO) UpdateWithOptimisticLock(ctx context.Context, walletID string, updateFn func(*dbmodel.Wallet) error) error {
	args := m.Called(ctx, walletID, updateFn)
	return args.Error(0)
}

func (m *MockWalletDAO) GetBalance(ctx context.Context, walletID string) (float64, error) {
	args := m.Called(ctx, walletID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockWalletDAO) ListAll(ctx context.Context) ([]dbmodel.Wallet, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dbmodel.Wallet), args.Error(1)
}

type MockAuditDAO struct {
	mock.Mock
}

func (m *MockAuditDAO) Create(ctx context.Context, log *dbmodel.AuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockAuditDAO) ListByWalletID(ctx context.Context, walletID string) ([]dbmodel.AuditLog, error) {
	args := m.Called(ctx, walletID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dbmodel.AuditLog), args.Error(1)
}

func (m *MockAuditDAO) GetSince(ctx context.Context, walletID string, since interface{}) ([]dbmodel.AuditLog, error) {
	args := m.Called(ctx, walletID, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dbmodel.AuditLog), args.Error(1)
}

func TestCreateWallet_Success(t *testing.T) {
	mockWalletDAO := new(MockWalletDAO)
	mockAuditDAO := new(MockAuditDAO)

	// Setup expectations
	mockWalletDAO.On("GetByUserID", mock.Anything, "test_user").Return(nil, nil)
	mockWalletDAO.On("Create", mock.Anything, mock.MatchedBy(func(w *dbmodel.Wallet) bool {
		return w.UserID == "test_user" && w.Status == "active"
	})).Return(nil)
	mockAuditDAO.On("Create", mock.Anything, mock.Anything).Return(nil)

	// Service uses mockWalletDAO, mockAuditDAO
	svc := service.NewWalletService(mockWalletDAO, mockAuditDAO)

	// Test
	resp, err := svc.CreateWallet(context.Background(), &service.CreateWalletRequest{
		UserID: "test_user",
	})

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "test_user", resp.UserID)
	assert.Equal(t, float64(0), resp.Balance)
	assert.Equal(t, "active", resp.Status)

	// Verify mocks were called
	mockWalletDAO.AssertExpectations(t)
	mockAuditDAO.AssertExpectations(t)
}

func TestCreateWallet_AlreadyExists(t *testing.T) {
	mockWalletDAO := new(MockWalletDAO)
	mockAuditDAO := new(MockAuditDAO)

	existingWallet := &dbmodel.Wallet{
		ID:     "wallet_123",
		UserID: "test_user",
		Status: "active",
	}

	mockWalletDAO.On("GetByUserID", mock.Anything, "test_user").Return(existingWallet, nil)

	svc := service.NewWalletService(mockWalletDAO, mockAuditDAO)

	resp, err := svc.CreateWallet(context.Background(), &service.CreateWalletRequest{
		UserID: "test_user",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	if domErr, ok := err.(*dbmodel.DomainError); ok {
		assert.Equal(t, dbmodel.ErrWalletAlreadyExists, domErr.Code)
	}

	mockWalletDAO.AssertExpectations(t)
}

func TestGetWallet_Success(t *testing.T) {
	mockWalletDAO := new(MockWalletDAO)
	mockAuditDAO := new(MockAuditDAO)

	wallet := &dbmodel.Wallet{
		ID:      "wallet_123",
		UserID:  "test_user",
		Balance: dbmodel.JSONQuery([]byte("500.00")),
		Status:  "active",
		Version: 1,
	}

	mockWalletDAO.On("GetByID", mock.Anything, "wallet_123").Return(wallet, nil)

	svc := service.NewWalletService(mockWalletDAO, mockAuditDAO)

	resp, err := svc.GetWallet(context.Background(), "wallet_123")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "wallet_123", resp.WalletID)
	assert.Equal(t, "test_user", resp.UserID)

	mockWalletDAO.AssertExpectations(t)
}

func TestGetWallet_NotFound(t *testing.T) {
	mockWalletDAO := new(MockWalletDAO)
	mockAuditDAO := new(MockAuditDAO)

	mockWalletDAO.On("GetByID", mock.Anything, "wallet_123").Return(nil, nil)

	svc := service.NewWalletService(mockWalletDAO, mockAuditDAO)

	resp, err := svc.GetWallet(context.Background(), "wallet_123")

	assert.Error(t, err)
	assert.Nil(t, resp)
	if domErr, ok := err.(*dbmodel.DomainError); ok {
		assert.Equal(t, dbmodel.ErrWalletNotFound, domErr.Code)
	}

	mockWalletDAO.AssertExpectations(t)
}

type MockTransactionDAO struct {
	mock.Mock
}

func (m *MockTransactionDAO) Create(ctx context.Context, transaction *dbmodel.Transaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

func (m *MockTransactionDAO) GetByID(ctx context.Context, transactionID string) (*dbmodel.Transaction, error) {
	args := m.Called(ctx, transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dbmodel.Transaction), args.Error(1)
}

func (m *MockTransactionDAO) GetByIdempotencyKey(ctx context.Context, walletID, key string) (*dbmodel.Transaction, error) {
	args := m.Called(ctx, walletID, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dbmodel.Transaction), args.Error(1)
}

func (m *MockTransactionDAO) ListByWalletID(ctx context.Context, walletID string, limit, offset int) ([]dbmodel.Transaction, int64, error) {
	args := m.Called(ctx, walletID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dbmodel.Transaction), args.Get(1).(int64), args.Error(2)
}

func (m *MockTransactionDAO) ListByWalletIDAndType(ctx context.Context, walletID, txType string, limit, offset int) ([]dbmodel.Transaction, int64, error) {
	args := m.Called(ctx, walletID, txType, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dbmodel.Transaction), args.Get(1).(int64), args.Error(2)
}

func (m *MockTransactionDAO) ListByWalletIDAndDateRange(ctx context.Context, walletID string, fromTime, toTime interface{}, limit, offset int) ([]dbmodel.Transaction, int64, error) {
	args := m.Called(ctx, walletID, fromTime, toTime, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]dbmodel.Transaction), args.Get(1).(int64), args.Error(2)
}

func (m *MockTransactionDAO) CountRecentTransactions(ctx context.Context, walletID string, windowMinutes int) (int64, error) {
	args := m.Called(ctx, walletID, windowMinutes)
	return args.Get(0).(int64), args.Error(1)
}

type MockLimitDAO struct {
	mock.Mock
}

func (m *MockLimitDAO) Create(ctx context.Context, limit *dbmodel.TransactionLimit) error {
	args := m.Called(ctx, limit)
	return args.Error(0)
}

func (m *MockLimitDAO) GetByWalletIDAndPeriod(ctx context.Context, walletID, period string) (*dbmodel.TransactionLimit, error) {
	args := m.Called(ctx, walletID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dbmodel.TransactionLimit), args.Error(1)
}

func (m *MockLimitDAO) Update(ctx context.Context, limit *dbmodel.TransactionLimit) error {
	args := m.Called(ctx, limit)
	return args.Error(0)
}

func (m *MockLimitDAO) UpdateAmount(ctx context.Context, limitID string, additionalAmount float64) error {
	args := m.Called(ctx, limitID, additionalAmount)
	return args.Error(0)
}

func (m *MockLimitDAO) ResetLimitIfNeeded(ctx context.Context, walletID, period string, resetTime interface{}) error {
	args := m.Called(ctx, walletID, period, resetTime)
	return args.Error(0)
}

func (m *MockLimitDAO) GetAllLimitsForWallet(ctx context.Context, walletID string) ([]dbmodel.TransactionLimit, error) {
	args := m.Called(ctx, walletID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dbmodel.TransactionLimit), args.Error(1)
}

func TestDepositValidation(t *testing.T) {
	mockWalletDAO := new(MockWalletDAO)
	mockTransactionDAO := new(MockTransactionDAO)
	mockLimitDAO := new(MockLimitDAO)
	mockAuditDAO := new(MockAuditDAO)

	cfg, _ := config.Load()
	fraudService := service.NewFraudService(mockTransactionDAO, cfg)
	limitService := service.NewLimitService(mockLimitDAO, cfg)

	mockWalletDAO.On("GetByID", mock.Anything, "wallet_123").Return(&dbmodel.Wallet{
		ID:      "wallet_123",
		Balance: dbmodel.JSONQuery([]byte("100.00")),
		Version: 0,
	}, nil)

	mockTransactionDAO.On("GetByIdempotencyKey", mock.Anything, "wallet_123", "key1").Return(nil, nil)
	mockTransactionDAO.On("CountRecentTransactions", mock.Anything, "wallet_123", mock.Anything).Return(int64(0), nil)
	mockLimitDAO.On("GetByWalletIDAndPeriod", mock.Anything, "wallet_123", "daily").Return(nil, nil)
	mockLimitDAO.On("GetByWalletIDAndPeriod", mock.Anything, "wallet_123", "weekly").Return(nil, nil)
	mockLimitDAO.On("Create", mock.Anything, mock.Anything).Return(nil)

	txService := service.NewTransactionService(mockWalletDAO, mockTransactionDAO, mockLimitDAO, mockAuditDAO, fraudService, limitService)

	// Test invalid amount
	resp, err := txService.Deposit(context.Background(), "wallet_123", &service.DepositRequest{
		Amount:         -50,
		IdempotencyKey: "key1",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	if domErr, ok := err.(*dbmodel.DomainError); ok {
		assert.Equal(t, dbmodel.ErrInvalidAmount, domErr.Code)
	}
}

func TestWithdraw_InsufficientBalance(t *testing.T) {
	mockWalletDAO := new(MockWalletDAO)
	mockTransactionDAO := new(MockTransactionDAO)
	mockLimitDAO := new(MockLimitDAO)
	mockAuditDAO := new(MockAuditDAO)

	cfg, _ := config.Load()
	fraudService := service.NewFraudService(mockTransactionDAO, cfg)
	limitService := service.NewLimitService(mockLimitDAO, cfg)

	mockWalletDAO.On("GetByID", mock.Anything, "wallet_123").Return(&dbmodel.Wallet{
		ID:      "wallet_123",
		Balance: dbmodel.JSONQuery([]byte("50.00")),
		Version: 0,
	}, nil)

	mockTransactionDAO.On("GetByIdempotencyKey", mock.Anything, "wallet_123", "key1").Return(nil, nil)

	txService := service.NewTransactionService(mockWalletDAO, mockTransactionDAO, mockLimitDAO, mockAuditDAO, fraudService, limitService)

	resp, err := txService.Withdraw(context.Background(), "wallet_123", &service.WithdrawRequest{
		Amount:         100,
		IdempotencyKey: "key1",
		OTP:            "123456",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	if domErr, ok := err.(*dbmodel.DomainError); ok {
		assert.Equal(t, dbmodel.ErrInsufficientBalance, domErr.Code)
	}
}
