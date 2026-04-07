package controller

import (
	"encoding/json"
	"net/http"

	"github.com/digital-wallet/internal/service"
)

type WalletController struct {
	walletService *service.WalletService
}

func NewWalletController(walletService *service.WalletService) *WalletController {
	return &WalletController{
		walletService: walletService,
	}
}

type CreateWalletReq struct {
	UserID string `json:"user_id"`
}

func (c *WalletController) CreateWallet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateWalletReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteValidationError(w, "Invalid request body")
		return
	}

	if req.UserID == "" {
		WriteValidationError(w, "user_id is required")
		return
	}

	resp, err := c.walletService.CreateWallet(ctx, &service.CreateWalletRequest{
		UserID: req.UserID,
	})

	if err != nil {
		statusCode := GetStatusCodeFromError(err)
		WriteErrorResponse(w, statusCode, err)
		return
	}

	WriteSuccessResponse(w, http.StatusCreated, resp)
}

func (c *WalletController) GetWallet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		WriteUnauthorizedError(w)
		return
	}

	resp, err := c.walletService.GetWalletByUserID(ctx, userID)
	if err != nil {
		statusCode := GetStatusCodeFromError(err)
		WriteErrorResponse(w, statusCode, err)
		return
	}

	WriteSuccessResponse(w, http.StatusOK, resp)
}
