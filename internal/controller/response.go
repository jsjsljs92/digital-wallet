package controller

import (
	"encoding/json"
	"net/http"

	"github.com/digital-wallet/internal/dbmodel"
)

type PaginationInfo struct {
	Total  int64 `json:"total"`
	Offset int   `json:"offset"`
	Limit  int   `json:"limit"`
	HasMore bool `json:"has_more"`
}

type DataResponse struct {
	Data       interface{}    `json:"data,omitempty"`
	Pagination *PaginationInfo `json:"pagination,omitempty"`
	Error      *ErrorResponse  `json:"error,omitempty"`
}

type ErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func WriteSuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := DataResponse{Data: data}
	json.NewEncoder(w).Encode(resp)
}

func WriteSuccessResponseWithPagination(w http.ResponseWriter, statusCode int, data interface{}, pagination *PaginationInfo) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := DataResponse{
		Data:       data,
		Pagination: pagination,
	}
	json.NewEncoder(w).Encode(resp)
}

func WriteErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errResp := &ErrorResponse{
		Code:    "INTERNAL_ERROR",
		Message: err.Error(),
		Details: make(map[string]interface{}),
	}

	if domainErr, ok := err.(*dbmodel.DomainError); ok {
		errResp.Code = string(domainErr.Code)
		errResp.Message = domainErr.Message
		errResp.Details = domainErr.Details
	}

	resp := DataResponse{Error: errResp}
	json.NewEncoder(w).Encode(resp)
}

func WriteValidationError(w http.ResponseWriter, message string) {
	WriteErrorResponse(w, http.StatusBadRequest,
		dbmodel.NewDomainError(
			dbmodel.ErrInvalidInput,
			message,
		))
}

func WriteUnauthorizedError(w http.ResponseWriter) {
	WriteErrorResponse(w, http.StatusUnauthorized,
		dbmodel.NewDomainError(
			dbmodel.ErrUnauthorized,
			"Unauthorized",
		))
}

func GetStatusCodeFromError(err error) int {
	if domainErr, ok := err.(*dbmodel.DomainError); ok {
		switch domainErr.Code {
		case dbmodel.ErrInsufficientBalance,
			dbmodel.ErrInvalidAmount,
			dbmodel.ErrInvalidInput:
			return http.StatusBadRequest
		case dbmodel.ErrWalletNotFound:
			return http.StatusNotFound
		case dbmodel.ErrWalletAlreadyExists,
			dbmodel.ErrDailyLimitExceeded,
			dbmodel.ErrWeeklyLimitExceeded,
			dbmodel.ErrOptimisticLockFailed,
			dbmodel.ErrIdempotencyConflict:
			return http.StatusConflict
		case dbmodel.ErrRateLimitExceeded:
			return http.StatusTooManyRequests
		case dbmodel.ErrUnauthorized:
			return http.StatusUnauthorized
		case dbmodel.ErrInvalidOTP:
			return http.StatusUnauthorized
		default:
			return http.StatusInternalServerError
		}
	}
	return http.StatusInternalServerError
}
