package middleware

import (
	"encoding/json"
	"net/http"
	"runtime/debug"

	"github.com/digital-wallet/internal/controller"
)

func ErrorHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				resp := controller.DataResponse{
					Error: &controller.ErrorResponse{
						Code:    "INTERNAL_SERVER_ERROR",
						Message: "An unexpected error occurred",
						Details: map[string]interface{}{
							"error": err,
							"stack": string(debug.Stack()),
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
