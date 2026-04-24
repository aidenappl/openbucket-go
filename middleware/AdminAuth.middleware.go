package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/aidenappl/openbucket-go/env"
	"github.com/aidenappl/openbucket-go/responder"
)

// AdminAuth validates that the request carries a valid admin bearer token.
func AdminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			responder.SendJSONError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			responder.SendJSONError(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(env.AdminToken)) != 1 {
			responder.SendJSONError(w, http.StatusForbidden, "invalid admin token")
			return
		}

		next.ServeHTTP(w, r)
	})
}
