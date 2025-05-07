package middleware

import (
	"net/http"
)

func Concurrency(next http.HandlerFunc, limit int64) http.HandlerFunc {
	limiter := make(chan struct{}, limit)
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case limiter <- struct{}{}:
			next.ServeHTTP(w, r)
			<-limiter
		default:
			http.Error(w, "try later", http.StatusServiceUnavailable)
		}
	}
}
