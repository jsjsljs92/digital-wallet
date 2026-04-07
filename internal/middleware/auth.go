package middleware

import (
	"context"
	"net/http"

	"github.com/digital-wallet/internal/config"
)

func Auth(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Auth.MockAuthEnabled {
				userID := r.Header.Get("X-User-ID")
				if userID == "" {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				ctx := context.WithValue(r.Context(), "user_id", userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// In production, validate JWT token here
			auth := r.Header.Get("Authorization")
			if auth == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Mock JWT validation
			ctx := context.WithValue(r.Context(), "user_id", "user_from_jwt")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
