// Package middleware provides HTTP middleware for the SD proxy.
package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// BearerAuth returns a middleware that validates Authorization: Bearer <token> headers.
// It uses constant-time comparison to prevent timing-based token enumeration attacks.
func BearerAuth(token string, next http.Handler) http.Handler {
	tokenBytes := []byte(token)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		provided := strings.TrimPrefix(authHeader, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(provided), tokenBytes) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
