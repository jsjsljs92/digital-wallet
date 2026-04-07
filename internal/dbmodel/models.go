package dbmodel

import (
	"time"

	"gorm.io/gorm"
)

type Wallet struct {
	ID        string         `gorm:"column:id;primaryKey;type:varchar(36)"`
	UserID    string         `gorm:"column:user_id;type:varchar(36);uniqueIndex;not null"`
	Balance   string         `gorm:"column:balance;type:decimal(19,2);default:0;not null"`
	Status    string         `gorm:"column:status;type:varchar(20);default:'active';not null"`
	Version   int64          `gorm:"column:version;type:bigint;default:0;not null"`
	CreatedAt time.Time      `gorm:"column:created_at;type:datetime;autoCreateTime:milli"`
	UpdatedAt time.Time      `gorm:"column:updated_at;type:datetime;autoUpdateTime:milli"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;type:datetime;index"`
}

func (Wallet) TableName() string {
	return "wallets"
}

type Transaction struct {
	ID             string         `gorm:"column:id;primaryKey;type:varchar(36)"`
	WalletID       string         `gorm:"column:wallet_id;type:varchar(36);index;not null"`
	Type           string         `gorm:"column:type;type:varchar(20);not null"` // deposit, withdraw
	Amount         string         `gorm:"column:amount;type:decimal(19,2);not null"`
	NewBalance     string         `gorm:"column:new_balance;type:decimal(19,2);not null"`
	Status         string         `gorm:"column:status;type:varchar(20);default:'completed';not null"`
	IdempotencyKey string         `gorm:"column:idempotency_key;type:varchar(255);index"`
	FraudDetected  bool           `gorm:"column:fraud_detected;type:boolean;default:false"`
	Reason         string         `gorm:"column:reason;type:text"`
	Metadata       string         `gorm:"column:metadata;type:json"`
	CreatedAt      time.Time      `gorm:"column:created_at;type:datetime;autoCreateTime:milli"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;type:datetime;autoUpdateTime:milli"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;type:datetime;index"`

	Wallet *Wallet `gorm:"foreignKey:WalletID;references:ID"`
}

func (Transaction) TableName() string {
	return "transactions"
}

type TransactionLimit struct {
	ID            string         `gorm:"column:id;primaryKey;type:varchar(36)"`
	WalletID      string         `gorm:"column:wallet_id;type:varchar(36);uniqueIndex:,composite:idx_wallet_period;not null"`
	Period        string         `gorm:"column:period;type:varchar(20);uniqueIndex:,composite:idx_wallet_period;not null"` // daily, weekly
	Amount        string         `gorm:"column:amount;type:decimal(19,2);default:0;not null"`
	Limit         string         `gorm:"column:limit;type:decimal(19,2);not null"`
	ResetAt       time.Time      `gorm:"column:reset_at;type:datetime;not null"`
	CreatedAt     time.Time      `gorm:"column:created_at;type:datetime;autoCreateTime:milli"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;type:datetime;autoUpdateTime:milli"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;type:datetime;index"`

	Wallet *Wallet `gorm:"foreignKey:WalletID;references:ID"`
}

func (TransactionLimit) TableName() string {
	return "transaction_limits"
}

type AuditLog struct {
	ID        string         `gorm:"column:id;primaryKey;type:varchar(36)"`
	WalletID  string         `gorm:"column:wallet_id;type:varchar(36);index;not null"`
	Action    string         `gorm:"column:action;type:varchar(100);not null"` // wallet_created, deposit, withdraw, etc
	OldValue  string         `gorm:"column:old_value;type:json"`
	NewValue  string         `gorm:"column:new_value;type:json"`
	Details   string         `gorm:"column:details;type:json"`
	CreatedAt time.Time      `gorm:"column:created_at;type:datetime;autoCreateTime:milli"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;type:datetime;index"`

	Wallet *Wallet `gorm:"foreignKey:WalletID;references:ID"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

type User struct {
	ID        string         `gorm:"column:id;primaryKey;type:varchar(36)"`
	Email     string         `gorm:"column:email;type:varchar(255);uniqueIndex;not null"`
	Name      string         `gorm:"column:name;type:varchar(255);not null"`
	Status    string         `gorm:"column:status;type:varchar(20);default:'active';not null"`
	CreatedAt time.Time      `gorm:"column:created_at;type:datetime;autoCreateTime:milli"`
	UpdatedAt time.Time      `gorm:"column:updated_at;type:datetime;autoUpdateTime:milli"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;type:datetime;index"`
}

func (User) TableName() string {
	return "users"
}

type IdempotencyRecord struct {
	ID             string         `gorm:"column:id;primaryKey;type:varchar(36)"`
	WalletID       string         `gorm:"column:wallet_id;type:varchar(36);uniqueIndex:,composite:idx_wallet_key;not null"`
	IdempotencyKey string         `gorm:"column:idempotency_key;type:varchar(255);uniqueIndex:,composite:idx_wallet_key;not null"`
	Response       string         `gorm:"column:response;type:json;not null"`
	StatusCode     int            `gorm:"column:status_code;type:int;not null"`
	CreatedAt      time.Time      `gorm:"column:created_at;type:datetime;autoCreateTime:milli"`
	ExpiresAt      time.Time      `gorm:"column:expires_at;type:datetime;index"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;type:datetime;index"`

	Wallet *Wallet `gorm:"foreignKey:WalletID;references:ID"`
}

func (IdempotencyRecord) TableName() string {
	return "idempotency_records"
}
