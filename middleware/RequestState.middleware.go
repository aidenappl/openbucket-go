package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

var HostIDContextKey contextKey = "hostID"
var RequestIDContextKey contextKey = "requestID"

func RequestState(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		hostID := uuid.New().String()

		requestID := uuid.New().String()

		ctx := context.WithValue(r.Context(), HostIDContextKey, hostID)
		ctx = context.WithValue(ctx, RequestIDContextKey, requestID)

		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func GetHostID(r *http.Request) string {
	if hostID, ok := r.Context().Value(HostIDContextKey).(string); ok {
		return hostID
	}
	return ""
}

func GetRequestID(r *http.Request) string {
	if requestID, ok := r.Context().Value(RequestIDContextKey).(string); ok {
		return requestID
	}
	return ""
}
