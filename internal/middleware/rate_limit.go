package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/digital-wallet/internal/config"
	"github.com/redis/go-redis/v9"
)

func RateLimit(redisClient *redis.Client, cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				next.ServeHTTP(w, r)
				return
			}

			key := fmt.Sprintf("rate_limit:%s:%d", userID, time.Now().Unix()/int64(cfg.RateLimit.Window))

			count, err := redisClient.Incr(ctx, key).Result()
			if err != nil {
				// If Redis fails, allow request
				next.ServeHTTP(w, r)
				return
			}

			if count == 1 {
				redisClient.Expire(ctx, key, time.Duration(cfg.RateLimit.Window)*time.Second)
			}

			if int(count) > cfg.RateLimit.Requests {
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.RateLimit.Requests))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", fmt.Sprintf("%d", cfg.RateLimit.Window))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.RateLimit.Requests))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", cfg.RateLimit.Requests-int(count)))

			next.ServeHTTP(w, r)
		})
	}
}
