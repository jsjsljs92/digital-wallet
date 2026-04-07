package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

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

	transactions, total, err := c.transactionService.GetTransactionHistory(ctx, walletResp.WalletID, limit, offset)
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
