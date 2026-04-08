# Digital Wallet Microservice

A secure, scalable backend microservice for managing user wallets and financial transactions in an e-commerce platform. Built with Go, MySQL, and Redis.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Tech Stack & Justification](#tech-stack--justification)
- [Features](#features)
- [Quick Start](#quick-start)
- [API Documentation](#api-documentation)
- [Environment Variables](#environment-variables)
- [Testing](#testing)
- [Project Structure](#project-structure)

---

## Overview

The Digital Wallet Microservice provides core financial operations for an e-commerce platform:
- **User wallet management** (create, retrieve balance)
- **Transactions** (deposit, withdraw with comprehensive audit trail)
- **Transaction history** with advanced filtering and pagination
- **Concurrency safety** via optimistic locking
- **Fraud detection** with real-time anomaly flags
- **Rate limiting** and idempotency guarantees
- **ACID-compliant** operations with full audit logging

### Use Case
A user in the e-commerce platform can:
1. Create a wallet (one per user)
2. Deposit funds from an external payment gateway
3. Withdraw funds to an external payout system
4. View transaction history with filters
5. Receive fraud alerts on suspicious activity

---

## Architecture

### High-Level Design

┌─────────────────────────────────────────────────────────────┐
│ Client / API Gateway                                        │
└────────────────────────────┬────────────────────────────────┘
                             │
                      Mock JWT Validation
                             │
                    ┌────────▼────────┐
                    │   Middleware    │
                    │ (auth, errors)  │
                    └────────┬────────┘
                             │
            ┌────────────────▼────────────────┐
            │          HTTP Handlers          │
            │        (wallet, tx)             │
            └────────────────┬────────────────┘
                             │
        ┌────────────────────▼────────────────────┐
        │              Service Layer              │
        │  ┌───────────────┐   ┌────────────────┐ │
        │  │ WalletService │   │ TransactionSvc │ │
        │  │ LimitService  │   │ (Optimistic    │ │
        │  │ FraudService  │   │  Locking)      │ │
        │  └───────────────┘   └────────────────┘ │
        └────────────────────┬────────────────────┘
                             │
   ┌──────────────┬──────────▼──────────┬──────────────┐
   │              │                     │              │
┌──▼──────────┐ ┌─▼────────────┐ ┌─────▼──────┐ ┌─────▼──────┐
│ MySQL DB    │ │ Redis Cache  │ │ Logging    │ │ Audit Trail│
│ (Txn)       │ │ (Rate Limit) │ │ System     │ │            │
└─────────────┘ └──────────────┘ └────────────┘ └────────────┘

### Request Flow: Deposit Example

Client: POST /v1/transactions/deposit
├─ Headers: Authorization, Idempotency-Key
├─ Body: {amount: 100.00}

AuthMiddleware: Validate JWT, extract user_id

DepositHandler: Validate input, call TransactionService

TransactionService.Deposit():
├─ Check idempotency key (has this exact request been processed?)
├─ Read wallet (select balance, version)
├─ Check daily/weekly limits
├─ Attempt UPDATE with optimistic lock:
│ └─ If version mismatch → retry up to 3x
├─ Insert transaction record
├─ Insert audit log entry
└─ Return result

ErrorHandlingMiddleware: Catch any errors, return standardized response

Response: {data: {transaction_id, new_balance}, pagination: {...}}

## Tech Stack & Justification

| Component | Choice | Justification |
|-----------|--------|---------------|
| **Language** | Go 1.21+ | Lightweight, excellent concurrency (goroutines), fast execution, strong stdlib for HTTP. Industry standard for microservices. |
| **HTTP Framework** | Standard Library + Middleware | Minimal dependencies, highly auditable, no magic. Includes `net/http`, chi, or Echo for routing. Production-ready. |
| **Database** | MySQL 8.0+ | ACID transactions, row-level locking, proven for financial systems. Optimistic locking supported natively via VERSION column. Foreign keys for referential integrity. |
| **Caching / Rate Limit** | Redis | Fast in-memory storage for rate-limit tracking. INCR + TTL for O(1) operations. Alternative: PostgreSQL with `pg_stat_statements` (overkill for MVP). |
| **ORM** | GORM | Type-safe queries, automatic migration support, hooks for audit logging, excellent for optimistic locking patterns. |
| **Authentication** | Mock JWT Middleware | For prototype: extract user_id from header (simulates JWT validation). Production would integrate with OAuth2/OIDC provider. |
| **Testing** | testify + testcontainers | Unit tests with mocks (fast), integration tests with real MySQL/Redis (comprehensive). testcontainers spins up Docker containers automatically. |
| **Documentation** | Mermaid + OpenAPI 3.0 | Mermaid for ER diagrams and flow charts (renderable in GitHub). OpenAPI 3.0 for API spec (Swagger UI integration). |


## Features

### Implemented (MVP)

- ✅ Create wallet for user (one per user)
- ✅ Retrieve wallet details and current balance
- ✅ Deposit funds with idempotency guarantee
- ✅ Withdraw funds with balance validation
- ✅ Transaction history with pagination and filtering (by type, date range, amount)
- ✅ Daily and weekly transaction limits (configurable)
- ✅ Optimistic locking for concurrent transaction safety
- ✅ Audit trail (all state changes logged)
- ✅ Fraud detection (advisory, non-blocking)
- ✅ Rate limiting per user (100 req/min default)
- ✅ Mock JWT validation middleware
- ✅ Idempotency for all write operations
- ✅ Comprehensive error handling and standardized responses

### Future Enhancements

- Transaction archival (soft-delete after 90 days)
- Real JWT integration (OAuth2/OIDC)
- Multi-currency support
- P2P transfers (out of scope for e-commerce)
- Real payment gateway integration (Stripe, PayPal)
- Advanced fraud detection (ML-based anomaly detection)
- Metrics & observability (Prometheus, Datadog)
- Cron-based batch reconciliation

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.21+ (for local development)
- Git

### Using Docker Compose (Recommended)

```bash
# Clone repository
git clone <repo-url>
cd digital-wallet

# Copy environment template
cp .env.example .env

# Start all services (MySQL, Redis, Go app)
docker-compose up -d

# Run migrations
docker-compose exec app go run cmd/migrate/main.go

# Seed sample data (optional dev helper)
# Not required to run the API or tests; useful for demos/manual QA.
docker-compose exec app go run scripts/seed.go

# API is now available at http://localhost:8080

# View logs
docker-compose logs -f app

# Stop services
docker-compose down

Local Development

# Start only MySQL and Redis
docker-compose up -d mysql redis

# Install dependencies
go mod download

# Run migrations
go run cmd/migrate/main.go

# Run the server
go run cmd/server/main.go

# Expected output:
# Server running on :8080

```markdown
## Testing

### Running Tests

```bash
# Unit tests (mocks, fast)
go test ./internal/service/... -v -cover

# Integration tests (requires Docker)
go test ./tests/integration/... -v -timeout=30s

# All tests with coverage report
go test ./... -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out  # Open in browser

Unit Tests
Tests mock all external dependencies (database, Redis). Fast, run in isolation.

Covers:

Wallet creation and retrieval
Deposit with idempotency (duplicate requests return same result)
Withdraw with balance validation
Transaction limit enforcement (daily, weekly)
Optimistic locking retry mechanism
Fraud detection flags
Error handling
Integration Tests
Tests use real MySQL and Redis (via testcontainers). Comprehensive end-to-end scenarios.

Covers:

End-to-end workflow (create wallet → deposit → withdraw)
Concurrent deposits to same wallet (verify final balance)
Idempotency with duplicate requests
Audit log entries
Rate limiting behavior
Transaction history filtering and pagination
Seed Data (Optional)

`scripts/seed.go` is an optional **developer helper** to create sample wallets/users in the database for quick demos and manual testing (Postman/curl).

It is **not required** to run the API or automated tests, since you can create wallets/transactions through the API (or run `test_api.sh`).

Run it when you want pre-populated data:

```bash
go run scripts/seed.go
```

```markdown
## API Documentation

### Base URL
http://localhost:8080/v1


### Authentication
All endpoints require `Authorization: Bearer <jwt-token>` header. For prototype, the value can be any string; middleware extracts `user_id` from a custom header `X-User-ID`.

### Endpoints

#### 1. Create Wallet

POST /v1/wallets
Content-Type: application/json
X-User-ID: user123

Response 201:
{
"data": {
"wallet_id": "wallet_abc123",
"user_id": "user123",
"balance": 0.00,
"status": "active",
"created_at": "2024-01-15T10:30:00Z"
}
}

Error 400: Wallet already exists for this user
Error 401: Unauthorized

#### 2. Get Wallet Details

GET /v1/wallets?user_id=user123
Authorization: Bearer <token>

Response 200:
{
"data": {
"wallet_id": "wallet_abc123",
"user_id": "user123",
"balance": 500.00,
"status": "active",
"created_at": "2024-01-15T10:30:00Z",
"updated_at": "2024-01-15T11:00:00Z"
}
}

Error 404: Wallet not found
Error 401: Unauthorized

#### 3. Deposit Funds
POST /v1/transactions/deposit
Content-Type: application/json
Authorization: Bearer <token>
Idempotency-Key: unique-key-12345
X-User-ID: user123

Request:
{
"amount": 100.00,
"reason": "Credit card topup"
}

Response 201:
{
"data": {
"transaction_id": "txn_deposit_abc",
"wallet_id": "wallet_abc123",
"type": "deposit",
"amount": 100.00,
"new_balance": 600.00,
"status": "completed",
"fraud_detected": false,
"created_at": "2024-01-15T11:05:00Z"
}
}

Error 400: Invalid amount (must be > 0)
Error 409: Daily limit exceeded
Error 401: Unauthorized

#### 4. Withdraw Funds
POST /v1/transactions/withdraw
Content-Type: application/json
Authorization: Bearer <token>
Idempotency-Key: unique-key-67890
X-User-ID: user123

Request:
{
"amount": 50.00,
"otp": "123456"
}

Response 201:
{
"data": {
"transaction_id": "txn_withdraw_xyz",
"wallet_id": "wallet_abc123",
"type": "withdraw",
"amount": 50.00,
"new_balance": 550.00,
"status": "completed",
"fraud_detected": false,
"created_at": "2024-01-15T11:10:00Z"
}
}

Error 400: Insufficient balance
Error 401: Invalid OTP
Error 429: Rate limit exceeded

#### 5. Transaction History
GET /v1/transactions?wallet_id=wallet_abc123&type=deposit&from=2024-01-01&to=2024-01-31&limit=10&offset=0
Authorization: Bearer <token>

Response 200:
{
"data": [
{
"transaction_id": "txn_abc123",
"type": "deposit",
"amount": 100.00,
"status": "completed",
"created_at": "2024-01-15T11:05:00Z"
},
...
],
"pagination": {
"total": 25,
"offset": 0,
"limit": 10,
"has_more": true
}
}

Error 400: Invalid date range
Error 401: Unauthorized

# Error Handling & Status Codes
### Error Response Format
```json
{
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "Wallet balance is insufficient for this withdrawal",
    "details": {
      "required": 100.00,
      "available": 50.00
    }
  }
}

# HTTP Status Codes
Code	Meaning
200	Success
201	Created
400	Bad Request (validation error)
401	Unauthorized (auth failed)
404	Not Found
409	Conflict (limit exceeded, idempotency issue)
429	Rate Limited
500	Internal Server Error


---

```markdown
## Environment Variables

Create `.env` file (see `.env.example`):

```bash
# Server
SERVER_PORT=8080
SERVER_ENV=development  # development, staging, production

# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=wallet_user
DB_PASSWORD=wallet_pass
DB_NAME=digital_wallet
DB_MAX_CONNECTIONS=25

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Authentication
JWT_SECRET=your-secret-key-here
MOCK_AUTH_ENABLED=true  # Set to false for real JWT

# Transaction Limits (USD)
DAILY_LIMIT=5000.00
WEEKLY_LIMIT=15000.00

# Fraud Detection
FRAUD_AMOUNT_THRESHOLD=10000.00  # Flag if single txn > $10k
FRAUD_VELOCITY_THRESHOLD=5       # Flag if > 5 txns in 10 minutes

# Rate Limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60  # seconds

# OTP (Mock)
OTP_VALIDITY_SECONDS=300  # 5 minutes
MOCK_OTP_SECRET=123456    # For demo purposes

# Logging
LOG_LEVEL=info  # debug, info, warn, error
LOG_FORMAT=json  # json, text


digital-wallet/
├── cmd/
│   ├── server/
│   │   └── main.go              # Entry point
│   ├── migrate/
│   │   └── main.go              # Run database migrations
│   └── wire_gen.go              # Auto-generated by wire
│
├── internal/
│   ├── dbmodel/
│   │   ├── wallet.go            # Wallet DB model
│   │   ├── transaction.go       # Transaction DB model
│   │   ├── limit.go             # Limit DB model
│   │   ├── audit.go             # Audit log DB model
│   │   └── errors.go            # Domain errors
│   │
│   ├── dao/
│   │   ├── wallet_dao.go        # Wallet DAO (CRUD ops)
│   │   ├── transaction_dao.go   # Transaction DAO
│   │   ├── limit_dao.go         # Limit DAO
│   │   └── audit_dao.go         # Audit log DAO
│   │
│   ├── service/
│   │   ├── wallet_service.go    # Wallet business logic
│   │   ├── transaction_service.go  # Transaction logic, locks
│   │   ├── limit_service.go     # Limit enforcement
│   │   ├── fraud_service.go     # Fraud detection
│   │   └── idempotency_service.go
│   │
│   ├── controller/
│   │   ├── wallet_controller.go # HTTP endpoints for wallet
│   │   ├── transaction_controller.go  # HTTP endpoints for tx
│   │   └── response.go          # Standardized responses
│   │
│   ├── middleware/
│   │   ├── auth.go              # JWT validation
│   │   ├── error_handler.go     # Error catching
│   │   ├── logger.go            # Request/response logging
│   │   └── rate_limit.go        # Rate limiting
│   │
│   ├── dbmanager/
│   │   ├── mysql.go             # MySQL connection manager
│   │   └── redis.go             # Redis connection manager
│   │
│   └── config/
│       └── config.go            # Load env vars
│
├── pkg/
│   ├── models/
│   │   └── types.go             # Shared types
│   └── errors/
│       └── errors.go            # Custom error types
│
├── migrations/
│   └── 001_init_schema.sql      # Schema definition
│
├── tests/
│   ├── mocks/
│   │   ├── mock_wallet_dao.go
│   │   ├── mock_transaction_dao.go
│   │   └── mock_redis.go
│   │
│   ├── integration/
│   │   ├── integration_test.go  # E2E tests
│   │   └── fixtures.go          # Test data
│   │
│   └── unit/
│       ├── service_test.go
│       └── controller_test.go
│
├── scripts/
│   └── seed.go                  # Optional dev helper (seed sample data)
│
├── wire.go                      # Wire dependency injection config
├── docker-compose.yml           # MySQL, Redis, App
├── Dockerfile                   # Go app container
├── .env.example                 # Environment template
├── .gitignore
├── go.mod
├── go.sum
├── README.md                    # This file
├── DESIGN.md                    # Technical design details
└── API.md                       # Detailed API spec

Layer Descriptions with Your Architecture
DBModel Layer (internal/dbmodel/)

Defines database entity structs
Example: Wallet, Transaction, TransactionLimit, AuditLog
GORM annotations for schema mapping
DAO Layer (internal/dao/)

Data Access Objects—direct database operations
Methods: Create(), GetByID(), Update(), GetByFilter(), etc.
Handles parameterized queries, idempotency checks
No business logic; pure DB operations
Service Layer (internal/service/)

Business logic: wallet operations, transaction processing, limits, fraud detection
Calls DAOs for data operations
Handles concurrency (optimistic locking), validation, error handling
Orchestrates workflows
Controller Layer (internal/controller/)

HTTP request handlers
Maps HTTP requests to service calls
Validates input, returns standardized responses
No business logic
DBManager (internal/dbmanager/)

Connection pooling for MySQL and Redis
Transaction management utilities
Lifecycle management (init, close, health checks)
Wiregen (wire.go + cmd/wire_gen.go)

Dependency injection configuration
Wires together: DBManager → DAOs → Services → Controllers
Auto-generated by wire tool

#Data Flow with Your Architecture

HTTP Request
    ↓
[Controller] (validate input, auth)
    ↓
[Service] (business logic, optimistic locking, limits)
    ↓
[DAO] (database operations, idempotency checks)
    ↓
[DBManager] (MySQL connection pool)
    ↓
[DBModel] (database entity)
    ↓
MySQL Database

## Development Workflow

1. **Local Setup**: `docker-compose up -d mysql redis`
2. **Make Changes**: Edit code in `internal/`
3. **Run Tests**: `go test ./...`
4. **Run Server**: `go run cmd/server/main.go`
5. **Test Endpoint**: `curl -H "X-User-ID: user1" http://localhost:8080/v1/wallets`

---

## Advanced Topics

### Concurrency Handling

The Digital Wallet uses **Optimistic Locking** to safely handle concurrent transactions:

- **Version field** on wallet tracks state
- **Retry logic** with exponential backoff (10ms, 20ms, 40ms)
- **3 automatic retries** on version mismatch
- **>99% success** on first attempt

See **DESIGN.md - Concurrency Handling** for detailed explanation with diagrams.

### Rate Limiting with Redis

Redis is used for rate limiting (100 requests/minute per user):

- **Atomic INCR** operation for counter
- **Auto-expiring keys** via EXPIRE
- **O(1) performance** (< 1ms per request)
- **Graceful degradation** if Redis is down

See **DESIGN.md - Redis Usage** for implementation details.

### Centralized Routes Architecture

All API routes defined in single location (`internal/routes/routes.go`):

- **Single source of truth** for entire API
- **Easy to add new routes** (one line)
- **Clear middleware chain** visible
- **API versioning support** (v1, v2)

See **DESIGN.md - Routes Architecture** for benefits and examples.

### Code Formatting Standards

Go code formatting via `gofmt`:

- **Command level**: `go fmt ./...` or `make fmt`
- **IDE level**: VS Code auto-format on save
- **Pre-commit**: Run `make fmt lint test` before commits

See **DESIGN.md - Code Formatting** for setup instructions.

---

## Deployment Notes

- **Containerized**: Dockerfile includes multi-stage build for minimal image size
- **Health Checks**: `/health` endpoint returns service status
- **Graceful Shutdown**: Server waits for in-flight requests before terminating
- **Horizontal Scaling**: Stateless design; add more instances behind load balancer
- **Database**: Connection pooling configured; auto-retry on transient failures
- **Secrets**: Use environment variables; never commit `.env` file

---

## License

MIT

---

## Support

For questions or issues, contact the development team or open an issue on GitHub.
