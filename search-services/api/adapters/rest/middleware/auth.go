package middleware

import (
	"net/http"
	"strings"
)

type TokenVerifier interface {
	Verify(token string) error
}

func Auth(next http.HandlerFunc, verifier TokenVerifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.Fields(r.Header.Get("Authorization"))
		if len(token) != 2 || token[0] != "Token" {
			http.Error(w, "bad authorization header", http.StatusUnauthorized)
			return
		}

		if err := verifier.Verify(token[1]); err != nil {
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return

		}

		next.ServeHTTP(w, r)

	}
}
