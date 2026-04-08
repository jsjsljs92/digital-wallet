package service

import (
	"context"
)

// FraudDetector abstracts fraud detection logic.
type FraudDetector interface {
	DetectFraud(ctx context.Context, walletID string, amount float64) bool
}

// LimitEnforcer abstracts transaction limit checks and updates.
type LimitEnforcer interface {
	CheckDailyLimit(ctx context.Context, walletID string, amount float64) error
	CheckWeeklyLimit(ctx context.Context, walletID string, amount float64) error
	UpdateDailyLimit(ctx context.Context, walletID string, amount float64) error
	UpdateWeeklyLimit(ctx context.Context, walletID string, amount float64) error
}
