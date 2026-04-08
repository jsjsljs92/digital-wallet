# Testing Guide - Digital Wallet

Complete guide to test the Digital Wallet microservice using various testing methods.

---

## Prerequisites

Install these tools before testing:

```bash
# Required
brew install docker          # Docker Desktop for containerization
brew install go              # Go compiler (v1.21+)
brew install jq              # JSON query tool for parsing API responses
brew install curl            # HTTP client (usually pre-installed)
brew install mysql-client    # MySQL command-line client
brew install redis           # Redis CLI for checking cache
```

**Verify installations:**
```bash
docker --version
go version
jq --version
curl --version
mysql --version
redis-cli --version
```

---

## Unit Testing

Run Go unit tests to verify business logic and error handling.

### Quick Test
```bash
make test
```

### Full Test with Coverage
```bash
make test-coverage
```

### Expected Output
```
✓ TestDomainError_ErrorCodes
✓ TestWalletService_CreateWallet
✓ TestWalletService_Deposit
✓ TestWalletService_Withdraw
✓ TestTransactionService_ListTransactions
✓ TestIdempotencyKey_Validation
```

---

## API Testing

### Prerequisites
Start the Docker stack:
```bash
make docker-up
```

Verify services are healthy:
```bash
docker compose ps
```

Expected output:
```
digital-wallet-mysql     Up (healthy)
digital-wallet-redis     Up (healthy)
digital-wallet-app       Up (running)
```

---

### Method 1: Automated Test Script (Recommended)

**Fastest way to test all endpoints.**

```bash
chmod +x test_api.sh
./test_api.sh
```

**What it tests:**
- ✅ Health check endpoint
- ✅ Create wallet
- ✅ Deposit funds
- ✅ Withdraw funds
- ✅ List transactions
- ✅ Transaction filtering
- ✅ Idempotency validation

**Time:** ~30 seconds | **Output:** Colored, formatted JSON responses

---

### Method 2: Manual Testing with curl

#### 2.1 Health Check
```bash
curl -s http://localhost:8080/v1/health -H "X-User-ID: your-user-id"
```

**Expected Response:**
{
  "status": "ok"
}

#### 2.2 Create Wallet (for user1)
```bash
curl -s -X POST http://localhost:8080/v1/wallets \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user1" \
  -H "Idempotency-Key: wallet-create-1" \
  -d '{"user_id":"user1"}' | jq '.data'
```

**Expected Response (201):**
```json
{
  "wallet_id": "wallet_abc123",
  "user_id": "user1",
  "balance": 0,
  "status": "active",
  "version": 0,
  "created_at": "2026-04-08 07:55:27 +0000 UTC",
  "updated_at": "2026-04-08 07:55:27 +0000 UTC"
}
```

#### 2.3 Get Wallet Details
```bash
curl -s -X GET http://localhost:8080/v1/wallets \
  -H "X-User-ID: user1" | jq '.data'
```

**Expected Response:**
```json
{
  "wallet_id": "wallet_abc123",
  "user_id": "user1",
  "balance": 0,
  "status": "active",
  "version": 0,
  "created_at": "2026-04-08 07:55:27 +0000 UTC",
  "updated_at": "2026-04-08 07:55:27 +0000 UTC"
}
```

#### 2.4 Deposit Funds
```bash
curl -s -X POST http://localhost:8080/v1/transactions/deposit \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user1" \
  -H "Idempotency-Key: deposit-1" \
  -d '{
    "amount": 100.00,
    "reason": "Initial deposit"
  }' | jq '.data'
```

**Expected Response (201):**
```json
{
  "transaction_id": "txn_001",
  "wallet_id": "wallet_abc123",
  "type": "deposit",
  "amount": 100,
  "new_balance": 100,
  "status": "completed",
  "fraud_detected": false,
  "created_at": "2026-04-08 07:55:27 +0000 UTC"
}
```

#### 2.5 Withdraw Funds
```bash
curl -s -X POST http://localhost:8080/v1/transactions/withdraw \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user1" \
  -H "Idempotency-Key: withdraw-1" \
  -d '{
    "amount": 30.00,
    "reason": "Withdrawal",
    "otp": "123456"
  }' | jq '.data'
```

**Expected Response (201):**
```json
{
  "transaction_id": "txn_002",
  "wallet_id": "wallet_abc123",
  "type": "withdraw",
  "amount": 30,
  "new_balance": 70,
  "status": "completed",
  "fraud_detected": false,
  "created_at": "2026-04-08 07:55:27 +0000 UTC"
}
```

#### 2.6 List Transactions
```bash
curl -s -X GET "http://localhost:8080/v1/transactions?limit=10&offset=0" \
  -H "X-User-ID: user1" | jq '.data'
```

**Expected Response:**
```json
[
  {
    "transaction_id": "txn_002",
    "wallet_id": "wallet_abc123",
    "type": "withdraw",
    "amount": 30,
    "new_balance": 70,
    "status": "completed",
    "fraud_detected": false,
    "created_at": "2026-04-08 07:55:28 +0000 UTC"
  },
  {
    "transaction_id": "txn_001",
    "wallet_id": "wallet_abc123",
    "type": "deposit",
    "amount": 100,
    "new_balance": 100,
    "status": "completed",
    "fraud_detected": false,
    "created_at": "2026-04-08 07:55:27 +0000 UTC"
  }
]
```

#### 2.7 Filter Transactions by Type
```bash
curl -s -X GET "http://localhost:8080/v1/transactions?type=deposit&limit=10" \
  -H "X-User-ID: user1" | jq '.data'
```

#### 2.8 Filter by Date Range
```bash
curl -s -X GET "http://localhost:8080/v1/transactions?from=2026-04-01&to=2026-04-08&limit=10" \
  -H "X-User-ID: user1" | jq '.data'
```

#### 2.9 Test Idempotency (Send Same Request Twice)
```bash
# First request
curl -s -X POST http://localhost:8080/v1/transactions/deposit \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user1" \
  -H "Idempotency-Key: duplicate-test" \
  -d '{"amount": 50.00, "reason": "Test"}' | jq '.data.transaction_id'

# Second request (same Idempotency-Key)
curl -s -X POST http://localhost:8080/v1/transactions/deposit \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user1" \
  -H "Idempotency-Key: duplicate-test" \
  -d '{"amount": 50.00, "reason": "Test"}' | jq '.data.transaction_id'

# Expected: Both return same response (no duplicate transaction)
```

---

### Method 3: Postman (Visual Testing)

#### Setup Postman Environment Variables
1. **Open Postman** → Click "Environments" (left sidebar)
2. **Create New** → Name: "Digital Wallet Dev"
3. **Add Variables:**
   | Variable | Value |
   |----------|-------|
   | base_url | http://localhost:8080 |
   | user_id | user1 |

4. **Select Environment** → Choose "Digital Wallet Dev" from dropdown

#### Create Collection Requests

**Request 1: Health Check**
- **Method:** GET
- **URL:** `{{base_url}}/v1/health`

**Request 2: Create Wallet**
- **Method:** POST
- **URL:** `{{base_url}}/v1/wallets`
- **Headers:**
  ```
  Content-Type: application/json
  X-User-ID: {{user_id}}
  Idempotency-Key: wallet-1
  ```
- **Body (raw JSON):**
  ```json
  {
    "user_id": "{{user_id}}"
  }
  ```

**Request 3: Deposit**
- **Method:** POST
- **URL:** `{{base_url}}/v1/transactions/deposit`
- **Headers:**
  ```
  Content-Type: application/json
  X-User-ID: {{user_id}}
  Idempotency-Key: deposit-1
  ```
- **Body (raw JSON):**
  ```json
  {
    "amount": 100.00,
    "reason": "Initial deposit"
  }
  ```

**Request 4: Withdraw**
- **Method:** POST
- **URL:** `{{base_url}}/v1/transactions/withdraw`
- **Headers:**
  ```
  Content-Type: application/json
  X-User-ID: {{user_id}}
  Idempotency-Key: withdraw-1
  ```
- **Body (raw JSON):**
  ```json
  {
    "amount": 30.00,
    "reason": "Withdrawal",
    "otp": "123456"
  }
  ```

**Request 5: List Transactions**
- **Method:** GET
- **URL:** `{{base_url}}/v1/transactions?limit=10&offset=0`
- **Headers:**
  ```
  X-User-ID: {{user_id}}
  ```

**Request 6: Filter by Type**
- **Method:** GET
- **URL:** `{{base_url}}/v1/transactions?type=deposit&limit=10`
- **Headers:**
  ```
  X-User-ID: {{user_id}}
  ```

---

## MySQL Data Verification

Verify data in MySQL database after running API tests.

### Method 1: Interactive MySQL Shell

```bash
docker exec -it digital-wallet-mysql mysql -u wallet_user -pwallet_pass -D digital_wallet
```

Then execute queries inside the shell:

```sql
-- View all wallets
SELECT id, user_id, balance, status, created_at FROM wallets;

-- View all transactions
SELECT id, wallet_id, type, amount, status, created_at FROM transactions;

-- View specific user's wallet
SELECT * FROM wallets WHERE user_id = 'testuser1';

-- Count transactions by type
SELECT type, COUNT(*) as count FROM transactions GROUP BY type;

-- View transaction details with new balance
SELECT id, wallet_id, type, amount, new_balance, status, created_at 
FROM transactions 
ORDER BY created_at DESC 
LIMIT 10;

-- Check idempotency keys (duplicates prevented)
SELECT wallet_id, idempotency_key, COUNT(*) as count 
FROM transactions 
GROUP BY wallet_id, idempotency_key 
HAVING count > 1;

-- Exit
exit
```

### Method 2: Quick Query (No Interactive Shell)

```bash
# List all wallets
docker exec digital-wallet-mysql mysql -u wallet_user -pwallet_pass -D digital_wallet -e "SELECT user_id, balance, status FROM wallets;"

# List recent transactions
docker exec digital-wallet-mysql mysql -u wallet_user -pwalket_pass -D digital_wallet -e "SELECT wallet_id, type, amount, new_balance, created_at FROM transactions ORDER BY created_at DESC LIMIT 10;"

# Count wallets
docker exec digital-wallet-mysql mysql -u wallet_user -pwalket_pass -D digital_wallet -e "SELECT COUNT(*) as wallet_count FROM wallets;"

# Check for duplicate idempotency keys
docker exec digital-wallet-mysql mysql -u wallet_user -pwalket_pass -D digital_wallet -e "SELECT wallet_id, idempotency_key, COUNT(*) as count FROM transactions GROUP BY wallet_id, idempotency_key HAVING count > 1;"
```

---

## Redis Cache & Rate Limiting Check

Verify rate limiting and cached data in Redis.

### Method 1: Interactive Redis CLI

```bash
docker exec -it digital-wallet-redis redis-cli
```

Then execute commands inside the shell:

```bash
# List all rate limit keys
KEYS rate_limit:testuser_1775634927

# Check rate limit for user1
GET rate_limit:user1:*

# View all keys in Redis
SCAN 0

# Get Redis info
INFO keyspace

# Check memory usage
INFO memory

# Count total keys
DBSIZE

# Exit
exit
```

*Key expires after 60s*

### Method 2: Quick Command (No Interactive Shell)

```bash
# List all rate limit entries
docker exec digital-wallet-redis redis-cli KEYS "rate_limit:*"

# Get detailed info about Redis
docker exec digital-wallet-redis redis-cli INFO

# Check how many keys are stored
docker exec digital-wallet-redis redis-cli DBSIZE

# Flush all data (use carefully!)
docker exec digital-wallet-redis redis-cli FLUSHALL
```

---

## Complete Testing Workflow

Run this complete workflow to test everything end-to-end:

```bash
# 1. Start services
make docker-up

# 2. Check services are healthy
docker compose ps

# 3. Run health check
curl -s http://localhost:8080/v1/health -H "X-User-ID: user1"

# 4. Run full automated API test
./test_api.sh

# 5. Verify data in MySQL
docker exec digital-wallet-mysql mysql -u wallet_user -pwalket_pass -D digital_wallet \
  -e "SELECT user_id, balance FROM wallets;"

# 6. Check rate limits in Redis
docker exec digital-wallet-redis redis-cli KEYS "rate_limit:*"

# 7. View application logs
docker compose logs -f app

# 8. Stop services (when done)
docker compose down
```

---

## Testing Error Scenarios

### Test Insufficient Balance
```bash
curl -X POST http://localhost:8080/v1/transactions/withdraw \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user1" \
  -H "Idempotency-Key: err-1" \
  -d '{"amount": 9999.00, "reason": "Should fail", "otp": "123456"}'

# Expected: 400 Bad Request
# Error: "INSUFFICIENT_BALANCE"
```

### Test Invalid Amount
```bash
curl -X POST http://localhost:8080/v1/transactions/deposit \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user1" \
  -H "Idempotency-Key: err-2" \
  -d '{"amount": -100.00, "reason": "Negative"}'

# Expected: 400 Bad Request
# Error: "INVALID_AMOUNT"
```

### Test Duplicate Wallet
```bash
# Create wallet first
curl -X POST http://localhost:8080/v1/wallets \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user1" \
  -H "Idempotency-Key: dup-wallet" \
  -d '{}'

# Try to create again with same user
curl -X POST http://localhost:8080/v1/wallets \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user1" \
  -H "Idempotency-Key: dup-wallet-2" \
  -d '{}'

# Expected: 400 Bad Request
# Error: "WALLET_ALREADY_EXISTS"
```

### Test Rate Limiting (100 requests/minute)
```bash
# Send 101 requests in quick succession
for i in {1..101}; do
  curl -X GET http://localhost:8080/v1/health \
    -H "X-User-ID: user1" \
    -w "Request $i: %{http_code}\n"
done

# Expected: 101st request returns 429 Too Many Requests
```

---

## Testing Key Features

### Concurrency Testing
Test optimistic locking with concurrent deposits:

```bash
# Terminal 1
curl -X POST http://localhost:8080/v1/transactions/deposit \
  -H "Content-Type: application/json" \
  -H "X-User-ID: concurrent-user" \
  -H "Idempotency-Key: con-1" \
  -d '{"amount": 50.00, "reason": "Concurrent 1"}' &

# Terminal 2
curl -X POST http://localhost:8080/v1/transactions/deposit \
  -H "Content-Type: application/json" \
  -H "X-User-ID: concurrent-user" \
  -H "Idempotency-Key: con-2" \
  -d '{"amount": 50.00, "reason": "Concurrent 2"}' &

# Terminal 3
curl -X POST http://localhost:8080/v1/transactions/deposit \
  -H "Content-Type: application/json" \
  -H "X-User-ID: concurrent-user" \
  -H "Idempotency-Key: con-3" \
  -d '{"amount": 50.00, "reason": "Concurrent 3"}'

# Expected: Final balance = $150.00 (all 3 deposits succeed)
# Check: SELECT * FROM wallets WHERE user_id = 'concurrent-user';
```

### Idempotency Testing
Send same request multiple times, should return same response:

```bash
IDEMPOTENCY_KEY="idempotent-test-1"
for i in {1..3}; do
  echo "Attempt $i:"
  curl -X POST http://localhost:8080/v1/transactions/deposit \
    -H "Content-Type: application/json" \
    -H "X-User-ID: idempotent-user" \
    -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
    -d '{"amount": 100.00, "reason": "Idempotent test"}' | jq .
  echo
done

# Expected: All 3 responses identical, only 1 transaction created
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `jq: command not found` | Run `brew install jq` |
| `docker compose not found` | Update Docker Desktop, or use `docker-compose` |
| `Connection refused (MySQL)` | Ensure `make docker-up` is running, check `docker compose ps` |
| `Connection refused (Redis)` | Same as MySQL, check service health |
| `400 Bad Request` | Check required headers: `X-User-ID`, `Idempotency-Key` for POST |
| `Invalid amount format` | Send `amount` as a JSON number (e.g. `100.00`), not a string |
| `Wallet already exists` | One wallet per user; try with different `X-User-ID` |
| `Insufficient balance` | Create wallet, deposit funds, then withdraw |

---

## Summary

| Method | Time | Best For |
|--------|------|----------|
| **Unit Tests** | ~10s | Testing business logic |
| **Automated Script** | ~30s | Quick full API validation |
| **Manual curl** | Varies | Learning & debugging specific endpoints |
| **Postman** | Varies | Professional testing & team collaboration |
| **MySQL Queries** | ~5s | Data verification |
| **Redis Queries** | ~2s | Rate limit & cache checking |

**Recommended workflow:** Unit Tests → Automated Script → Manual Testing → Data Verification
