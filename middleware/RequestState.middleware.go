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
		// Generate HostId
		hostID := uuid.New().String()
		// Generate RequestId
		requestID := uuid.New().String()
		// Store the IDs in the context
		ctx := context.WithValue(r.Context(), HostIDContextKey, hostID)
		ctx = context.WithValue(ctx, RequestIDContextKey, requestID)
		// Create a new request with the updated context
		r = r.WithContext(ctx)
		// Call the next handler in the chain
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
