-- Users table
CREATE TABLE IF NOT EXISTS users (
  id VARCHAR(36) PRIMARY KEY,
  email VARCHAR(255) UNIQUE NOT NULL,
  name VARCHAR(255) NOT NULL,
  status VARCHAR(20) DEFAULT 'active' NOT NULL,
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3),
  KEY idx_deleted_at (deleted_at)
);

-- Wallets table with version for optimistic locking
CREATE TABLE IF NOT EXISTS wallets (
  id VARCHAR(36) PRIMARY KEY,
  user_id VARCHAR(36) UNIQUE NOT NULL,
  balance DECIMAL(19,2) DEFAULT 0 NOT NULL,
  status VARCHAR(20) DEFAULT 'active' NOT NULL,
  version BIGINT DEFAULT 0 NOT NULL,
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  KEY idx_user_id (user_id),
  KEY idx_deleted_at (deleted_at)
);

-- Transactions table with idempotency key
CREATE TABLE IF NOT EXISTS transactions (
  id VARCHAR(36) PRIMARY KEY,
  wallet_id VARCHAR(36) NOT NULL,
  type VARCHAR(20) NOT NULL,
  amount DECIMAL(19,2) NOT NULL,
  new_balance DECIMAL(19,2) NOT NULL,
  status VARCHAR(20) DEFAULT 'completed' NOT NULL,
  idempotency_key VARCHAR(255),
  fraud_detected BOOLEAN DEFAULT FALSE,
  reason TEXT,
  metadata JSON,
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3),
  FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE CASCADE,
  KEY idx_wallet_id (wallet_id),
  KEY idx_idempotency_key (idempotency_key),
  KEY idx_deleted_at (deleted_at),
  UNIQUE KEY idx_wallet_idempotency (wallet_id, idempotency_key)
);

-- Transaction limits table
CREATE TABLE IF NOT EXISTS transaction_limits (
  id VARCHAR(36) PRIMARY KEY,
  wallet_id VARCHAR(36) NOT NULL,
  period VARCHAR(20) NOT NULL,
  amount DECIMAL(19,2) DEFAULT 0 NOT NULL,
  limit DECIMAL(19,2) NOT NULL,
  reset_at DATETIME NOT NULL,
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3),
  FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE CASCADE,
  UNIQUE KEY idx_wallet_period (wallet_id, period),
  KEY idx_deleted_at (deleted_at)
);

-- Audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
  id VARCHAR(36) PRIMARY KEY,
  wallet_id VARCHAR(36) NOT NULL,
  action VARCHAR(100) NOT NULL,
  old_value JSON,
  new_value JSON,
  details JSON,
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3),
  FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE CASCADE,
  KEY idx_wallet_id (wallet_id),
  KEY idx_deleted_at (deleted_at)
);

-- Idempotency records table
CREATE TABLE IF NOT EXISTS idempotency_records (
  id VARCHAR(36) PRIMARY KEY,
  wallet_id VARCHAR(36) NOT NULL,
  idempotency_key VARCHAR(255) NOT NULL,
  response JSON NOT NULL,
  status_code INT NOT NULL,
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
  expires_at DATETIME,
  deleted_at DATETIME(3),
  FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE CASCADE,
  UNIQUE KEY idx_wallet_key (wallet_id, idempotency_key),
  KEY idx_expires_at (expires_at),
  KEY idx_deleted_at (deleted_at)
);
