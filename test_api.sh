#!/bin/bash

set -e

API="http://localhost:8080"
USER="testuser_$(date +%s)"
IDKEY=$(date +%s)

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║             DIGITAL WALLET API TESTING SCRIPT                 ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""
echo "API Endpoint: $API"
echo "Test User: $USER"
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}1️⃣  Health Check${NC}"
curl -s $API/v1/health -H "X-User-ID: $USER" | jq '.'
echo ""

echo -e "${BLUE}2️⃣  Creating Wallet${NC}"
WALLET_RESPONSE=$(curl -s -X POST $API/v1/wallets \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER" \
  -H "Idempotency-Key: wallet-create" \
  -d "{\"user_id\": \"$USER\"}")
WALLET_ID=$(echo $WALLET_RESPONSE | jq '.data.wallet_id' -r)
echo $WALLET_RESPONSE | jq '.data'
echo "Wallet ID: $WALLET_ID"
echo ""

echo -e "${BLUE}3️⃣  Get Wallet Details${NC}"
curl -s -X GET "$API/v1/wallets" \
  -H "X-User-ID: $USER" | jq '.data'
echo ""

echo -e "${BLUE}4️⃣  Deposit \$100${NC}"
DEPOSIT_RESPONSE=$(curl -s -X POST $API/v1/transactions/deposit \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER" \
  -H "Idempotency-Key: dep-$IDKEY" \
  -d "{\"amount\": 100.00, \"reason\": \"Initial deposit\"}")
echo $DEPOSIT_RESPONSE | jq '.data'
echo ""

echo -e "${BLUE}5️⃣  Withdraw \$30${NC}"
curl -s -X POST $API/v1/transactions/withdraw \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER" \
  -H "Idempotency-Key: with-$IDKEY" \
  -d "{\"amount\": 30.00, \"reason\": \"Test withdrawal\", \"otp\": \"123456\"}" | jq '.data'
echo ""

echo -e "${BLUE}6️⃣  List All Transactions${NC}"
curl -s -X GET "$API/v1/transactions?wallet_id=$WALLET_ID&limit=10" \
  -H "X-User-ID: $USER" | jq '.data'
echo ""

echo -e "${BLUE}7️⃣  Filter by Transaction Type (Deposit)${NC}"
curl -s -X GET "$API/v1/transactions?wallet_id=$WALLET_ID&type=deposit&limit=10" \
  -H "X-User-ID: $USER" | jq '.data'
echo ""

echo -e "${BLUE}8️⃣  Final Wallet Balance${NC}"
curl -s -X GET "$API/v1/wallets" \
  -H "X-User-ID: $USER" | jq '.data.balance'
echo ""

echo -e "${BLUE}9️⃣  Test Idempotency${NC}"
echo "Request 1 (new transaction):"
IDEMPOTENT_KEY="idempotent-test-$IDKEY"
curl -s -X POST $API/v1/transactions/deposit \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER" \
  -H "Idempotency-Key: $IDEMPOTENT_KEY" \
  -d "{\"amount\": 50.00, \"reason\": \"Idempotency test\"}" | jq '.data.transaction_id'

echo "Request 2 (same key - should return same transaction):"
curl -s -X POST $API/v1/transactions/deposit \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER" \
  -H "Idempotency-Key: $IDEMPOTENT_KEY" \
  -d "{\"amount\": 50.00, \"reason\": \"Idempotency test\"}" | jq '.data.transaction_id'
echo ""

echo "✅ API Testing Complete!"
