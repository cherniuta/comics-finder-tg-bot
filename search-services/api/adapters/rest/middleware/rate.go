package middleware

import (
	"net/http"

	"go.uber.org/ratelimit"
)

func Rate(next http.HandlerFunc, rps int) http.HandlerFunc {
	limiter := ratelimit.New(rps)
	return func(w http.ResponseWriter, r *http.Request) {
		limiter.Take()
		next.ServeHTTP(w, r)

	}
}
