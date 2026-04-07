package middleware

import (
	"context"
	"log"
	"net/http"
	"time"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		log.Printf(
			"%s %s %s %s",
			r.RemoteAddr,
			r.Method,
			r.RequestURI,
			r.UserAgent(),
		)

		next.ServeHTTP(w, r)

		log.Printf(
			"Completed %s %s in %v",
			r.Method,
			r.RequestURI,
			time.Since(start),
		)
	})
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		ctx := context.WithValue(r.Context(), "request-id", requestID)
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-request"
}
