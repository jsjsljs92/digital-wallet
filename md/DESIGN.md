# Digital Wallet Microservice - Technical Design

This document details the database schema, concurrency strategy, error handling, security, and scalability considerations for the Digital Wallet microservice.

## Table of Contents

- [Database Schema](#database-schema)
- [Concurrency and Idempotency Strategy](#concurrency-and-idempotency-strategy)
- [Error Handling & Retry Strategy](#error-handling--retry-strategy)
- [Security Considerations](#security-considerations)
- [Scalability & Future Extensibility Plan](#scalability--future-extensibility-plan)

## Database Schema

### ER Diagram
┌──────────────────┐ ┌──────────────────┐
│ users │ │ wallets │
├──────────────────┤ ├──────────────────┤
│ user_id (PK) │◄────────│ wallet_id (PK) │
│ created_at │ 1:1 │ user_id (FK) │
│ email │ │ balance │
│ status │ │ version (INT) │
└──────────────────┘ │ status │
│ created_at │
│ updated_at │
└────────┬─────────┘
│
1:N
│
┌───────────────────┼───────────────────┐
│ │ │
┌─────────▼────────┐ ┌──────▼──────────┐ ┌────▼─────────┐
│ transactions │ │ trans_limits │ │ audit_logs │
├──────────────────┤ ├─────────────────┤ ├──────────────┤
│ txn_id (PK) │ │ limit_id (PK) │ │ log_id (PK) │
│ wallet_id (FK) │ │ user_id (FK) │ │ wallet_id(FK)│
│ type │ │ type │ │ txn_id (FK) │
│ amount │ │ period_start │ │ old_balance │
│ status │ │ cumulative_amt │ │ new_balance │
│ idempotency_key │ │ limit_threshold │ │ operation │
│ created_at │ │ created_at │ │ timestamp │
│ metadata_json │ └─────────────────┘ │ user_id │
└──────────────────┘ └──────────────┘

Key Indexes:

wallets: (user_id) UNIQUE, (created_at)
transactions: (wallet_id, idempotency_key) UNIQUE, (wallet_id, created_at), (type)
trans_limits: (user_id, type, period_start) UNIQUE
audit_logs: (wallet_id, created_at), (user_id, created_at)


### Table Definitions

#### wallets
```sql
CREATE TABLE wallets (
    wallet_id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL UNIQUE,
    balance DECIMAL(19,2) NOT NULL DEFAULT 0.00,
    version INT NOT NULL DEFAULT 1,
    status ENUM('active', 'frozen', 'closed') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    CONSTRAINT balance_non_negative CHECK (balance >= 0.00),
    INDEX idx_created_at (created_at)
);

Fields:

wallet_id: UUID, unique identifier for wallet
user_id: External user ID from identity service; one wallet per user (UNIQUE)
balance: Current balance in USD; DECIMAL for precision (not float)
version: Optimistic lock counter; incremented on every update
status: Wallet state (active, frozen for manual restriction, closed for deletion)
created_at, updated_at: Audit timestamps


transactions
CREATE TABLE transactions (
    txn_id VARCHAR(36) PRIMARY KEY,
    wallet_id VARCHAR(36) NOT NULL,
    type ENUM('deposit', 'withdraw') NOT NULL,
    amount DECIMAL(19,2) NOT NULL,
    status ENUM('pending', 'completed', 'failed', 'cancelled') DEFAULT 'pending',
    idempotency_key VARCHAR(255) NOT NULL,
    reason VARCHAR(255),
    fraud_detected BOOLEAN DEFAULT FALSE,
    metadata_json JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT amount_positive CHECK (amount > 0.00),
    UNIQUE KEY unique_idempotency (wallet_id, idempotency_key),
    FOREIGN KEY (wallet_id) REFERENCES wallets(wallet_id),
    INDEX idx_wallet_created (wallet_id, created_at),
    INDEX idx_type (type),
    INDEX idx_fraud (fraud_detected)
);

Fields:

txn_id: Unique transaction identifier
wallet_id: Foreign key linking to wallet
type: deposit or withdraw
amount: Transaction amount (positive; direction indicated by type)
status: Transaction state (pending → completed, or failed/cancelled)
idempotency_key: Client-provided unique key; prevents duplicate processing
fraud_detected: Boolean flag (advisory only; doesn't block transaction)
metadata_json: Extensible field for context (e.g., bank details, gateway reference)
Unique Constraint:
The (wallet_id, idempotency_key) UNIQUE index ensures that within a wallet, the same idempotency key produces exactly one transaction. On duplicate requests, the query returns the existing transaction.

transaction_limits
CREATE TABLE transaction_limits (
    limit_id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL,
    type ENUM('daily', 'weekly') NOT NULL,
    period_start DATE NOT NULL,
    cumulative_amount DECIMAL(19,2) NOT NULL DEFAULT 0.00,
    limit_threshold DECIMAL(19,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_user_period (user_id, type, period_start),
    INDEX idx_period_start (period_start)
);

Fields:

user_id: User identifier
type: daily or weekly limit
period_start: Start date of the limit window
cumulative_amount: Sum of all transactions (deposits + withdrawals) in this period
limit_threshold: Configured maximum from environment variables

audit_logs (Insert-Only)
CREATE TABLE audit_logs (
    log_id VARCHAR(36) PRIMARY KEY,
    wallet_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(50) NOT NULL,
    txn_id VARCHAR(36),
    operation_type ENUM('deposit', 'withdraw', 'manual_adjustment') NOT NULL,
    old_balance DECIMAL(19,2),
    new_balance DECIMAL(19,2) NOT NULL,
    amount_changed DECIMAL(19,2),
    metadata_json JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (wallet_id) REFERENCES wallets(wallet_id),
    FOREIGN KEY (txn_id) REFERENCES transactions(txn_id),
    INDEX idx_wallet_created (wallet_id, created_at),
    INDEX idx_user_created (user_id, created_at)
);

Fields:

Immutable record of every balance change
old_balance, new_balance: Before/after snapshot for reconciliation
metadata_json: Additional context (reason, fraud flags, etc.)
No UPDATE or DELETE allowed; insert-only for compliance


---

## **Chunk 3: Database Schema - Part 2**

```markdown
### users (Minimal)

```sql
CREATE TABLE users (
    user_id VARCHAR(50) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

Note: Users are managed by external identity service. This table is minimal and exists mainly for referential integrity. Can be denormalized with additional fields from auth system if needed.

Indexing Strategy
High-frequency queries:

wallets.user_id: Lookup wallet by user (most common read)
transactions.wallet_id + created_at: Filter transaction history with date range
transaction_limits.user_id + type + period_start: Check daily/weekly limits
audit_logs.wallet_id + created_at: Retrieve audit trail
Why these indexes matter:

O(log N) lookups instead of O(N) full table scans
Query planner chooses best path automatically
Regular monitoring: slow query log, EXPLAIN ANALYZE
Data Types & Precision
Monetary Values (DECIMAL):

All balances and amounts use DECIMAL(19,2) (not float)
Rationale: Floating-point arithmetic loses precision for financial calculations
Example: 0.1 + 0.2 = 0.30000000000000004 (float), 0.30 (decimal) ✓
DECIMAL(19,2): up to 19 digits total, 2 after decimal = up to $999,999,999,999,999.99
JSON Fields:

transactions.metadata_json: Store extensible data (bank details, gateway references)
audit_logs.metadata_json: Additional context for compliance
Benefit: No schema migration needed for new fields
Constraints
Data Integrity:

wallets.balance >= 0.00: CHECK constraint prevents negative balances
transactions.amount > 0.00: All transactions must have positive amounts
wallets.user_id UNIQUE: One wallet per user
(wallet_id, idempotency_key) UNIQUE: Prevent duplicate transactions
Foreign keys: Cascading deletes handled carefully (soft-delete preferred)
Referential Integrity:

Transactions reference valid wallets
Audit logs reference transactions and wallets
No orphaned records



---

## **Chunk 4: Concurrency and Idempotency Strategy**

```markdown
## Concurrency and Idempotency Strategy

### Problem: Lost Updates in Concurrent Transactions

**Scenario:**
Two concurrent deposit requests to same wallet with balance $100:
- Request 1: Read balance (100), increment to 150
- Request 2: Read balance (100), increment to 150
- **Result**: Balance is 150 instead of 200 (lost update)

### Solution: Optimistic Locking

**Mechanism:**
Each wallet has a `version` column (INT). On update:

1. **Read with Version:**
   ```sql
   SELECT balance, version FROM wallets WHERE wallet_id = 'wallet_123';
   -- Returns: balance=100, version=5

2. Attempt Atomic Update:
UPDATE wallets 
SET balance = 150, version = 6
WHERE wallet_id = 'wallet_123' AND version = 5;

Check Result:

If rows_affected == 1: Success; version match, update applied
If rows_affected == 0: Failure; version mismatch (another thread updated)
Why It Works:

No database locks; prevents deadlocks
Scales horizontally; no global lock contention
Version mismatch detected immediately
Concurrent Deposit Example:

Thread A                          Thread B
─────────────────────────────────────────────────
Read: balance=100, v=5
                                  Read: balance=100, v=5
UPDATE ... WHERE v=5 ✓
(rows affected: 1)
balance=150, v=6
                                  UPDATE ... WHERE v=5 ✗
                                  (rows affected: 0)
                                  Retry:
                                  Sleep 10ms
                                  Read: balance=150, v=6
                                  UPDATE ... WHERE v=6 ✓
                                  balance=200, v=7

Retry Logic
Service Implementation:
maxRetries = 3
baseBackoff = 100ms

for attempt := 0; attempt < maxRetries; attempt++:
    wallet = GetWallet(walletID)  // Read balance + version
    newBalance = wallet.balance + amount
    
    success = UpdateBalance(walletID, newBalance, wallet.version)
    
    if success:
        return Success  // ✓ Updated
    
    if OptimisticLockError:
        if attempt < maxRetries - 1:
            backoff = baseBackoff * 2^attempt + random jitter
            sleep(backoff)
            continue  // Retry
        else:
            return Error("Max retries exceeded")
    
    return Error  // Unrecoverable error

Expected retry rate: < 1% under normal conditions
Max retry count: 3 (configurable)
Backoff progression: 100ms → 200ms → 400ms (plus jitter ±20%)


Trade-offs:

✅ Avoids deadlocks; scales well
✅ Retry latency typically < 500ms
❌ Not suitable for extremely high-contention scenarios (1000s txns/sec on single wallet)
Problem: Duplicate Requests
Scenario:
Client retries deposit after network timeout:

Request 1: Deposit $50 → balance $150 ✓
Network timeout; client retries
Request 2 (duplicate): Deposit $50 → balance $200 ✗ (should still be $150)
Solution: Idempotency Keys
Mechanism:
Client provides unique idempotency_key per request. Database constraint ensures one transaction per key.

Check Idempotency:
SELECT txn_id, status FROM transactions 
WHERE wallet_id = 'wallet_123' AND idempotency_key = 'key_abc';


If found: Return cached result (don't re-execute)
If not: Proceed with transaction

Atomic Insert:
INSERT INTO transactions (txn_id, wallet_id, idempotency_key, ...) 
VALUES ('txn_xyz', 'wallet_123', 'key_abc', ...);

UNIQUE constraint on (wallet_id, idempotency_key) ensures exactly one
If duplicate key: Constraint violation → query for existing transaction

Return Result:
First request: New transaction created; balance updated; result returned
Retry: Existing transaction found; cached result returned; no duplicate balance change

Example:
Request 1: POST /v1/transactions/deposit
Headers: Idempotency-Key: "key_abc123"
Body: {amount: 50.00}

1. Check: SELECT * FROM transactions WHERE wallet_id='w1' AND idempotency_key='key_abc123'
   → No result
2. Process: Deposit logic
3. Insert: INSERT INTO transactions (txn_id, wallet_id, idempotency_key=key_abc123, ...)
4. Return: {txn_id: txn_001, new_balance: 150}

Request 2 (Retry, same Idempotency-Key):
1. Check: SELECT * FROM transactions WHERE wallet_id='w1' AND idempotency_key='key_abc123'
   → Found: txn_id=txn_001, status=completed
2. No re-execution
3. Return: {txn_id: txn_001, new_balance: 150}  ← Same result

Idempotency Key Format:

Recommended: UUID or {user_id}_{endpoint}_{timestamp}_{random}
Lifetime: Store indefinitely (or 90 days for archived transactions)
Client responsibility: Generate unique key; reuse on retry

Concurrency + Idempotency Together
Combined Flow:
1. Idempotency Check
   → Duplicate request? Return cached result ✓

2. Service Execution (with optimistic locking)
   → Concurrent updates? Retry with version increment ✓

3. Result
   → No lost updates (optimistic locking)
   → No duplicate processing (idempotency)

Example: Concurrent Requests with Idempotency
Request A (Deposit $50, key_abc)
Request B (Deposit $30, key_def)  ← Different idempotency key
Request A Retry (same key_abc)    ← Same idempotency key

Thread A1: Idempotency check (key_abc) → Not found
           Read wallet (balance=100, v=5)
           Attempt update v=5 → v=6, balance=150
           
Thread B:  Idempotency check (key_def) → Not found
           Read wallet (balance=100, v=5)  ← Reads old version
           Attempt update v=5 → FAIL (v already 6)
           Retry → Read (balance=150, v=6)
                   Attempt update v=6 → v=7, balance=180
                   
Thread A2: Idempotency check (key_abc) → Found txn_abc
           Return cached result (balance was 150)

Final balance: 180 ✓ (50 + 30 correctly applied)
No duplicates: Requests A1 and A2 processed once ✓


---

## ** Error Handling & Retry Strategy**

```markdown
## Error Handling & Retry Strategy

### Error Categories

#### 1. Client Errors (4xx) — No Automatic Retry

| Error | HTTP | Cause | Action |
|-------|------|-------|--------|
| Invalid Amount | 400 | amount <= 0 or > max | Client fixes input; retry manually |
| Insufficient Balance | 400 | balance < withdrawal amount | User deposits first; manual retry |
| Limit Exceeded | 409 | daily/weekly cumulative > threshold | User waits; retry after period resets |
| Wallet Not Found | 404 | wallet_id doesn't exist | User creates wallet first; retry with new wallet_id |
| Invalid OTP | 401 | OTP wrong or expired | User re-requests OTP; retry |
| Unauthorized | 401 | JWT invalid or missing | User re-authenticates; retry with valid token |
| Rate Limited | 429 | Too many requests (>100/min) | Client waits 60s; exponential backoff on retry |

**Client Retry Policy:**
- **No automatic retry** for 400, 401, 404 (client must fix root cause)
- **Exponential backoff** for 429 (60s, 120s, 300s)
- **Idempotency-Key required** for safe retries on write operations

#### 2. Server Errors (5xx) — Automatic Retry with Backoff

| Error | HTTP | Cause | Retry? | Strategy |
|-------|------|-------|--------|----------|
| Database Connection Lost | 500 | MySQL down, network timeout | Yes | 3 retries, exponential backoff |
| Optimistic Lock Retry Exhausted | 500 | Extreme contention (3+ retries failed) | Yes | 1 more attempt after 500ms |
| Audit Log Write Failed | 500 | Disk full, DB error | Yes | Critical; block operation until success |
| Redis Connection Lost | 500 | Cache unavailable | Yes (graceful degradation) | Rate limiting disabled; operations continue |

**Server Retry Policy:**
- **Automatic internal retries**: Up to 3 attempts for transient errors
- **Exponential backoff**: 100ms → 200ms → 400ms (plus jitter ±20%)
- **Circuit breaker** (optional): If 5+ failures in 60s, fail-fast for 30s
- **Idempotency-Key**: All retries use same key → same transaction

### Response Format

**Success Response:**
```json
HTTP 200 / 201
{
  "data": {
    "transaction_id": "txn_abc123",
    "wallet_id": "wallet_123",
    "type": "deposit",
    "amount": 100.00,
    "new_balance": 500.00,
    "status": "completed",
    "fraud_detected": false,
    "created_at": "2024-01-15T10:30:00Z"
  },
  "pagination": {
    "total": 50,
    "offset": 0,
    "limit": 10,
    "has_more": true
  }
}


Error Response:
HTTP 400 / 409 / 500
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "trace_id": "req_abc123xyz",
    "details": {
      "field": "amount",
      "constraint": "gt=0",
      "available": 50.00,
      "required": 100.00
    }
  }
}

Error Codes
Code	HTTP	Meaning
INVALID_INPUT	400	Validation failed (amount ≤ 0, missing fields)
INSUFFICIENT_BALANCE	400	Not enough funds for withdrawal
LIMIT_EXCEEDED	409	Daily/weekly limit exceeded
WALLET_NOT_FOUND	404	Wallet doesn't exist
INVALID_OTP	401	OTP invalid or expired
UNAUTHORIZED	401	JWT validation failed
RATE_LIMIT_EXCEEDED	429	Too many requests
IDEMPOTENCY_CONFLICT	409	Different amounts for same idempotency key
CONCURRENT_UPDATE_FAILED	500	Optimistic lock retry exhausted
INTERNAL_ERROR	500	Unrecoverable server error
Retry Logic (Service Layer)
Optimistic Locking Retry:


Retry Logic (Service Layer)
Optimistic Locking Retry:

func (svc *TransactionService) Deposit(ctx context.Context, ...) {
    const maxRetries = 3
    const baseBackoff = 100ms
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        wallet, err := WalletDAO.GetByID(walletID)
        if err != nil {
            return err  // Unrecoverable error
        }
        
        newBalance := wallet.Balance + amount
        
        err := WalletDAO.UpdateBalance(walletID, newBalance, wallet.Version)
        if err == nil {
            return success  // ✓ Balance updated
        }
        
        if err == OptimisticLockError {
            if attempt < maxRetries - 1 {
                backoff := baseBackoff * time.Duration(math.Pow(2, float64(attempt)))
                jitter := time.Duration(rand.Intn(int(backoff / 5)))
                time.Sleep(backoff + jitter)
                continue  // Retry
            }
        }
        
        return err  // Unrecoverable error
    }
    
    return Error("Deposit failed after max retries")
}


Database Error Handling
Connection Errors:

Timeout connecting to MySQL: Retry with backoff; if persistent, return 503 Service Unavailable
Connection pool exhausted: Wait for idle connection (timeout after 10s); return 503 if timeout
Query Errors:

Constraint violation: Return 409 Conflict (business logic error)
Syntax error: Return 500 Internal Server Error (code bug)
Row lock timeout (pessimistic): Return 500, retry on client
Logging Strategy
What to log:

Transaction ID, wallet ID, user ID (for tracing)
Request type (deposit/withdraw), amount, status
Error code and message (if applicable)
Execution time and retry count
What NOT to log:

Full OTP or sensitive credentials
Full JWT tokens
Complete request/response bodies (log only relevant fields)


Example Log Entry:
{
  "timestamp": "2024-01-15T10:30:45Z",
  "level": "INFO",
  "trace_id": "req_abc123xyz",
  "user_id": "user456",
  "wallet_id": "wallet_789",
  "operation": "deposit",
  "amount": 100.00,
  "status": "completed",
  "execution_time_ms": 125,
  "retries": 1,
  "fraud_detected": false
}



---

## **Security Considerations**

```markdown
## Security Considerations

### 1. Authentication & Authorization

**Current (Prototype):**
- Mock JWT validation via `X-User-ID` header (no cryptographic verification)
- No token signature validation
- **For Production**: Integrate with OAuth2/OIDC provider (Auth0, Okta, etc.)

**Per-Request Validation:**

All endpoints require Authorization header
Authorization: Bearer <jwt_token>

Middleware validates:

Token format (Bearer <token>)
User ID extraction from token claims
Token expiration
Signature verification (in production)


**Authorization (Per-Wallet):**
- Users can only access their own wallet
- Validate: `request.user_id == wallet.user_id`
- Return 403 Forbidden if mismatch

### 2. Input Validation

**All user inputs validated before database operations:**

| Field | Validation |
|-------|-----------|
| amount | > 0, ≤ 999999.99 (sanity check), DECIMAL precision |
| idempotency_key | Not empty, max 255 chars |
| reason | Optional, max 255 chars |
| date_from, date_to | Valid date format; from ≤ to |
| limit, offset | Positive integers; limit ≤ 100 |
| otp | 6 digits, numeric only |

**Implementation:**
- Use struct tags for validation (GORM, validator library)
- Reject before database call
- Return 400 Bad Request with details

### 3. OTP / Two-Factor Authentication (Withdrawals)

**Withdrawal requires OTP verification:**

POST /v1/transactions/withdraw
Request body: {amount: 50.00, otp: "123456"}

Validation:

OTP must be 6 digits
OTP must match user's current session OTP (expires after 5 minutes)
If invalid: Return 401 Unauthorized
If expired: Return 401 with "OTP expired" message
Production Implementation:

SMS/email OTP provider (Twilio, AWS SNS)
Store OTP hash in cache (not plaintext)
Rate limit OTP requests (3 requests per hour)

**Mock OTP (Prototype):**
Fixed secret: "123456" (for demo)
Validity: 5 minutes
Rate limit: Disabled (for testing)


### 4. Sensitive Data Protection

**At Rest:**
- Database passwords: Environment variables only (not hardcoded)
- Sensitive columns: Can use database-level encryption (TDE)
- Backups: Encrypted at rest and in transit

**In Transit:**
- HTTPS only (TLS 1.2+)
- Enforce via reverse proxy / load balancer
- HSTS header (HTTP Strict-Transport-Security)
- No HTTP fallback

**Logging:**
✗ Don't log: Full amounts, OTPs, JWT tokens, PII
✓ Do log: Transaction ID, type, status, user_id (anonymized last 4 digits)

Example safe log:
{
"txn_id": "txn_abc",
"operation": "deposit",
"status": "completed",
"user_id_last4": "****4567",
"timestamp": "2024-01-15T10:30:00Z"
}

### 5. SQL Injection Prevention

**All queries use parameterized statements (bound parameters):**

✓ Safe (Parameterized):
db.Where("wallet_id = ? AND user_id = ?", walletID, userID).First(&wallet)

✗ Unsafe (String Concatenation):
query := "SELECT * FROM wallets WHERE wallet_id = '" + walletID + "'"


**GORM automatically handles parameterization:**
- Bind variables passed separately from query
- Database driver escapes special characters
- No SQL injection possible

### 6. Fraud Detection (Advisory)

**Real-Time Checks (Non-Blocking):**
- Amount threshold: Flag if > $10,000
- Velocity check: Flag if > 5 transactions in 10 minutes
- Return fraud flag in response; don't block transaction

**Implementation:**
On each transaction:

Check if amount > FRAUD_AMOUNT_THRESHOLD
Query: Count transactions in last 10 minutes
If count > FRAUD_VELOCITY_THRESHOLD: Set fraud_detected=true
Log fraud flag to audit_logs
Continue transaction (advisory only)
In production, integrate with:

ML-based anomaly detection
External fraud detection service
Manual review queue for flagged transactions


### 7. Rate Limiting

**Per-User Rate Limit:**
Limit: 100 requests per minute (configurable)
Storage: Redis with INCR + TTL

Implementation:

key = "user:{user_id}:requests"
redis.INCR(key) → increment counter
redis.EXPIRE(key, 60) → set 60-second TTL
If counter > 100: Return 429 Too Many Requests
Graceful degradation:

If Redis down: Rate limiting disabled (warning logged)
Operations continue normally



### 8. CSRF Protection

**Not applicable** (API design)
- No session cookies used
- Token-based auth (JWT in Authorization header)
- Safe against CSRF by design

### 9. Data Breach Response

**In production:**
- Encryption of sensitive fields (AES-256)
- Regular security audits and penetration testing
- Incident response plan and breach notification procedures
- PII data minimization (store only necessary user info)
- Audit logs for compliance (SOC 2, PCI DSS if handling payment cards)


## Scalability & Future Extensibility Plan

### Current Architecture (MVP)

**Deployment Model:**
┌──────────────────┐
│ Load Balancer │
└────────┬─────────┘
│
┌────┴────┬─────────┬─────────┐
│ │ │ │
┌───▼──┐ ┌───▼──┐ ┌───▼──┐ ┌───▼──┐
│ App1 │ │ App2 │ │ App3 │ │ App4 │ (Stateless)
└───┬──┘ └───┬──┘ └───┬──┘ └───┬──┘
│ │ │ │
└────────┼────────┼────────┘
│
┌────▼─────────────┐
│ MySQL (Master) │ (Single instance, replicas for read scaling)
└────────┬─────────┘
│
┌───────┴───────┐
│ │
┌────▼───┐ ┌───▼────┐
│Replica1│ │Replica2│ (Read-only)
└────────┘ └────────┘
│
┌────▼───────────┐
│ Redis Cluster │ (For rate limiting, caching)
└────────────────┘


**Performance Targets (MVP):**
| Metric | Target |
|--------|--------|
| P50 Latency | < 50ms |
| P99 Latency | < 200ms |
| Availability | 99.9% |
| Throughput | 1,000 transactions/sec |

### Scaling Bottlenecks & Solutions

#### 1. Single MySQL Instance
**Current:** Handles ~1,000 txns/sec with R/W on master

**Bottleneck at Scale:** Single master becomes write bottleneck beyond 5,000 txns/sec

**Solution: Database Sharding (5,000 - 50,000 txns/sec)**

Shard wallets by hash(wallet_id) % num_shards

Wallet ID Database
0...Z → Shard 1 (MySQL-1)
A...M → Shard 2 (MySQL-2)
N...Z → Shard 3 (MySQL-3)

Router Logic:
func getShardForWallet(walletID string) *DB {
hash := crc32.ChecksumIEEE([]byte(walletID))
shardIndex := hash % numShards
return shards[shardIndex]
}

Benefits:

Write load distributed across shards
Scales linearly with shard count
Each shard: ~1,000-5,000 txns/sec
Considerations:

Cross-shard transactions difficult (mostly unnecessary)
Joins across shards not supported
Rebalancing on shard addition complex


#### 2. Redis for Rate Limiting
**Current:** Single Redis instance; INCR + TTL operations O(1)

**Bottleneck at Scale:** Single Redis becomes bottleneck beyond 50,000 requests/sec

**Solution: Redis Cluster**
Redis Cluster (3-6 nodes):

Automatic replication
Horizontal scalability
Failover support
config:
REDIS_CLUSTER=true
REDIS_NODES=redis1:6379,redis2:6379,redis3:6379

Benefits:

Each node handles subset of keys
Scales to millions of requests/sec
Built-in failover


#### 3. Application Tier
**Current:** Stateless Go services; horizontally scalable

**Scaling:** Add app instances behind load balancer
docker-compose scale app=10 # Scale to 10 instances

Load Balancer distributes traffic:

Round-robin
Least-connection
Health checks (/health endpoint)
No state on app nodes → easy horizontal scaling


#### 4. Read Replicas for Wallet Lookups
**Current:** All reads/writes on master

**Optimization:** Send reads to replicas
GetWalletByUserID(userID) → Query replica
Deposit (write) → Query master
ListTransactions (read) → Query replica

Benefits:

Reduces load on master
Better read throughput
Eventual consistency acceptable (< 1s lag)
Tradeoff:

Replica lag: Master → Replica ~ 100-500ms
Stale reads possible (acceptable for non-critical queries)

---

## Concurrency Handling - Optimistic Locking

### Problem: Race Conditions

Without concurrency control, concurrent updates to the same wallet can cause lost updates:

```
Wallet balance: $100
Thread A: Deposit $50 → Read: $100, Update: $150
Thread B: Deposit $30 → Read: $100, Update: $130
Result: ✗ Balance = $130 (WRONG! Should be $180)
```

### Solution: Optimistic Locking with Version Field

**Mechanism:**
1. **Read** wallet with current version
2. **Calculate** new balance
3. **Update** ONLY if version matches:
   ```sql
   UPDATE wallet
   SET balance = ?, version = version + 1
   WHERE id = ? AND version = ?
   ```
4. **Check** if update succeeded (RowsAffected == 1)
5. **Retry** if version mismatch (exponential backoff)

**Configuration:**
- Max retries: 3
- Initial wait: 10ms
- Backoff factor: 2x exponential (10ms → 20ms → 40ms)
- Success rate: >99% on first attempt
- Total retry time: < 100ms

**Implementation:** `internal/service/transaction_service.go`, `internal/dao/wallet_dao.go`

### Why Optimistic Locking?

| Aspect | Optimistic | Pessimistic | Distributed Locks |
|--------|-----------|-------------|------------------|
| Concurrency | High | Low | Depends on impl |
| Deadlock risk | None | High | Yes |
| Complexity | Simple | Moderate | Complex |
| Scalability | Excellent | Poor | Fair |
| Best for | Low contention | High contention | External sync |

---

## Redis Usage - Rate Limiting

### Configuration

**File:** `.env.example`

```bash
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
RATE_LIMIT_REQUESTS=100      # 100 requests
RATE_LIMIT_WINDOW=60         # per 60 seconds
```

### How It Works

**Middleware:** `internal/middleware/rate_limit.go`

**Key format:** `rate_limit:user_id:timestamp_window`

**Operations:**
1. `INCR` - Increment counter (O(1), atomic)
2. `EXPIRE` - Set TTL on key (O(1))

**Response headers:**
- `X-RateLimit-Limit`: 100
- `X-RateLimit-Remaining`: 85
- `Retry-After`: 45

### Performance

| Metric | Value |
|--------|-------|
| Time per request | < 1ms |
| Complexity | O(1) |
| Scalability | Millions of requests/sec |
| Thread-safe | ✅ Yes (atomic operations) |

### Graceful Degradation

If Redis is unavailable:
- Rate limiting is **disabled**
- API continues to work normally
- No cascading failures

---

## Routes Architecture - Centralized Management

### Design Pattern

**Before (Scattered):**
- Route definitions in controller files
- Each controller has `RegisterRoutes()` method
- Difficult to see complete API at a glance

**After (Centralized):**
- All routes in single file: `internal/routes/routes.go`
- Single function: `SetupRoutes()`
- Easy to understand complete API structure

### Implementation

**File:** `internal/routes/routes.go`

```go
func SetupRoutes(router chi.Router, cfg *config.Config, db *gorm.DB, redisClient *redis.Client) {
    // 1. Initialize dependencies (DAOs → Services → Controllers)
    walletDAO := dao.NewWalletDAO(db)
    walletService := service.NewWalletService(walletDAO, auditDAO)
    walletController := controller.NewWalletController(walletService)
    
    // 2. Apply global middleware
    router.Use(middleware.ErrorHandler)
    router.Use(middleware.Logger)
    router.Use(middleware.RequestID)
    router.Use(middleware.RateLimit(redisClient, cfg))
    router.Use(middleware.Auth(cfg))
    
    // 3. Define all routes
    router.Route("/v1", func(r chi.Router) {
        r.Get("/health", healthHandler)
        r.Post("/wallets", walletController.CreateWallet)
        r.Get("/wallets", walletController.GetWallet)
        r.Post("/transactions/deposit", transactionController.Deposit)
        r.Post("/transactions/withdraw", transactionController.Withdraw)
        r.Get("/transactions", transactionController.ListTransactions)
    })
}
```

**Integration in main:** `cmd/server/main.go`

```go
routes.SetupRoutes(router, cfg, db, redis)  // One line!
```

### Benefits

✅ **Single source of truth** for all routes
✅ **Easier to add features** (one line in routes.go)
✅ **No main.go changes** needed
✅ **Easy to implement versioning** (v1, v2 routes)
✅ **Route-specific middleware** can be added easily

### Adding New Routes

1. Create handler in controller:
   ```go
   func (c *WalletController) GetBalance(w http.ResponseWriter, r *http.Request) {
       // implementation
   }
   ```

2. Add one line to `internal/routes/routes.go`:
   ```go
   r.Get("/wallets/{id}/balance", walletController.GetBalance)
   ```

3. Done! Main.go remains unchanged.

---

## Code Formatting Standards

### Official Standard: gofmt

Go provides `gofmt` as the official formatter:

```bash
# Format a single file
go fmt filename.go

# Format all files in project
go fmt ./...

# Using Makefile
make fmt
```

**Advantages:**
- Official Go standard
- Consistent across all machines
- Required for CI/CD pipelines
- No configuration needed
- Part of Go toolchain

### IDE Integration: VS Code Auto-Format

**Setup (one-time):**

1. Install Go extension for VS Code
2. Edit `.vscode/settings.json`:
   ```json
   {
     "editor.formatOnSave": true,
     "[go]": {
       "editor.defaultFormatter": "golang.go",
       "editor.formatOnSave": true,
       "editor.codeActionsOnSave": {
         "source.organizeImports": true
       }
     }
   }
   ```
3. Restart VS Code

Now every save automatically formats code.

### Best Practice Workflow

**During development:**
- IDE auto-format on save (immediate feedback)
- Catch formatting issues immediately

**Before committing:**
```bash
make fmt        # Format all Go code
make lint       # Run go vet
make test       # Run tests
```

Or as one command:
```bash
make all        # fmt + lint + test + build
```

### Formatting Rules Go Enforces

- **Indentation**: Always tabs (1 tab = 8 spaces visually)
- **Line length**: Typically ≤ 100 characters
- **Brace style**: Opening brace on same line
- **Spacing**: Around operators and after keywords
- **Blank lines**: Between functions and logical blocks
- **Imports**: Alphabetical, grouped by stdlib/third-party

### Example

**Before:**
```go
func Create(ctx context.Context, req *Request) (*Response, error) {
  wallet := &Wallet{ID: "test", Status: "active"}
  err := dao.Create(ctx, wallet)
  if err != nil {return nil, err}
  return &Response{ID: wallet.ID}, nil
}
```

**After (go fmt):**
```go
func Create(ctx context.Context, req *Request) (*Response, error) {
	wallet := &Wallet{ID: "test", Status: "active"}
	err := dao.Create(ctx, wallet)
	if err != nil {
		return nil, err
	}
	return &Response{ID: wallet.ID}, nil
}
```

**Changes:**
- Indentation: Tabs instead of spaces
- Brace positioning: Moved to new line in if
- Spacing: Proper around operators