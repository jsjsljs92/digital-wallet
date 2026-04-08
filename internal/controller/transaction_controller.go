package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/digital-wallet/internal/service"
)

type TransactionController struct {
	transactionService *service.TransactionService
	walletService      *service.WalletService
}

func NewTransactionController(transactionService *service.TransactionService, walletService *service.WalletService) *TransactionController {
	return &TransactionController{
		transactionService: transactionService,
		walletService:      walletService,
	}
}

type DepositReq struct {
	Amount float64 `json:"amount"`
	Reason string  `json:"reason"`
}

type WithdrawReq struct {
	Amount float64 `json:"amount"`
	OTP    string  `json:"otp"`
}

func (c *TransactionController) Deposit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		WriteUnauthorizedError(w)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")

	// Get wallet
	walletResp, err := c.walletService.GetWalletByUserID(ctx, userID)
	if err != nil {
		statusCode := GetStatusCodeFromError(err)
		WriteErrorResponse(w, statusCode, err)
		return
	}

	var req DepositReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteValidationError(w, "Invalid request body")
		return
	}

	if req.Amount <= 0 {
		WriteValidationError(w, "Amount must be greater than 0")
		return
	}

	resp, err := c.transactionService.Deposit(ctx, walletResp.WalletID, &service.DepositRequest{
		Amount:         req.Amount,
		IdempotencyKey: idempotencyKey,
		Reason:         req.Reason,
	})

	if err != nil {
		statusCode := GetStatusCodeFromError(err)
		WriteErrorResponse(w, statusCode, err)
		return
	}

	WriteSuccessResponse(w, http.StatusCreated, resp)
}

func (c *TransactionController) Withdraw(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		WriteUnauthorizedError(w)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")

	// Get wallet
	walletResp, err := c.walletService.GetWalletByUserID(ctx, userID)
	if err != nil {
		statusCode := GetStatusCodeFromError(err)
		WriteErrorResponse(w, statusCode, err)
		return
	}

	var req WithdrawReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteValidationError(w, "Invalid request body")
		return
	}

	if req.Amount <= 0 {
		WriteValidationError(w, "Amount must be greater than 0")
		return
	}

	resp, err := c.transactionService.Withdraw(ctx, walletResp.WalletID, &service.WithdrawRequest{
		Amount:         req.Amount,
		IdempotencyKey: idempotencyKey,
		OTP:            req.OTP,
	})

	if err != nil {
		statusCode := GetStatusCodeFromError(err)
		WriteErrorResponse(w, statusCode, err)
		return
	}

	WriteSuccessResponse(w, http.StatusCreated, resp)
}

func (c *TransactionController) GetTransactions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		WriteUnauthorizedError(w)
		return
	}

	// Get wallet
	walletResp, err := c.walletService.GetWalletByUserID(ctx, userID)
	if err != nil {
		statusCode := GetStatusCodeFromError(err)
		WriteErrorResponse(w, statusCode, err)
		return
	}

	// Optional: wallet_id query param (must match authenticated user's wallet)
	if qWalletID := r.URL.Query().Get("wallet_id"); qWalletID != "" && qWalletID != walletResp.WalletID {
		WriteValidationError(w, "wallet_id does not match authenticated user")
		return
	}

	limit := 10
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	var filters service.TransactionHistoryFilters

	if txType := r.URL.Query().Get("type"); txType != "" {
		filters.Type = &txType
	}

	if from := r.URL.Query().Get("from"); from != "" {
		t, err := parseTimeParam(from)
		if err != nil {
			WriteValidationError(w, "invalid 'from' time (use RFC3339 or YYYY-MM-DD)")
			return
		}
		filters.FromTime = &t
	}

	if to := r.URL.Query().Get("to"); to != "" {
		t, err := parseTimeParam(to)
		if err != nil {
			WriteValidationError(w, "invalid 'to' time (use RFC3339 or YYYY-MM-DD)")
			return
		}
		filters.ToTime = &t
	}

	if minAmt := r.URL.Query().Get("min_amount"); minAmt != "" {
		v, err := strconv.ParseFloat(minAmt, 64)
		if err != nil {
			WriteValidationError(w, "invalid 'min_amount'")
			return
		}
		filters.MinAmount = &v
	}

	if maxAmt := r.URL.Query().Get("max_amount"); maxAmt != "" {
		v, err := strconv.ParseFloat(maxAmt, 64)
		if err != nil {
			WriteValidationError(w, "invalid 'max_amount'")
			return
		}
		filters.MaxAmount = &v
	}

	transactions, total, err := c.transactionService.GetTransactionHistoryFiltered(ctx, walletResp.WalletID, filters, limit, offset)
	if err != nil {
		WriteErrorResponse(w, GetStatusCodeFromError(err), err)
		return
	}

	pagination := &PaginationInfo{
		Total:   total,
		Offset:  offset,
		Limit:   limit,
		HasMore: int64(offset+limit) < total,
	}

	WriteSuccessResponseWithPagination(w, http.StatusOK, transactions, pagination)
}

func parseTimeParam(raw string) (time.Time, error) {
	// RFC3339 (e.g. 2026-04-08T07:55:28Z)
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}

	// Date only (YYYY-MM-DD) interpreted as UTC start-of-day.
	if d, err := time.Parse("2006-01-02", raw); err == nil {
		return d.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("invalid time format: %q", raw)
}
