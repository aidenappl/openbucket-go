package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aidenappl/openbucket-go/env"
	"github.com/aidenappl/openbucket-go/responder"
)

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestAdminAuth_ValidToken(t *testing.T) {
	env.AdminToken = "test-admin-token-12345"

	handler := AdminAuth(http.HandlerFunc(okHandler))
	req := httptest.NewRequest(http.MethodGet, "/admin/credentials", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token-12345")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestAdminAuth_MissingHeader(t *testing.T) {
	env.AdminToken = "test-admin-token-12345"

	handler := AdminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/credentials", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}

	var errResp responder.JSONError
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp.Success {
		t.Fatal("expected success=false")
	}
	if errResp.ErrorMessage != "missing authorization header" {
		t.Fatalf("expected 'missing authorization header', got '%s'", errResp.ErrorMessage)
	}
}

func TestAdminAuth_WrongToken(t *testing.T) {
	env.AdminToken = "test-admin-token-12345"

	handler := AdminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/credentials", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestAdminAuth_BadFormat(t *testing.T) {
	env.AdminToken = "test-admin-token-12345"

	handler := AdminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/credentials", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAdminAuth_CaseInsensitiveBearer(t *testing.T) {
	env.AdminToken = "test-admin-token-12345"

	handler := AdminAuth(http.HandlerFunc(okHandler))

	for _, prefix := range []string{"bearer", "BEARER", "Bearer", "bEaReR"} {
		t.Run(prefix, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin/credentials", nil)
			req.Header.Set("Authorization", prefix+" test-admin-token-12345")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200 for prefix %q, got %d", prefix, rr.Code)
			}
		})
	}
}

func TestAdminAuth_EmptyToken(t *testing.T) {
	env.AdminToken = "test-admin-token-12345"

	handler := AdminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/credentials", nil)
	req.Header.Set("Authorization", "Bearer ")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestAdminAuth_OnlySchemeNoToken(t *testing.T) {
	env.AdminToken = "test-admin-token-12345"

	handler := AdminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/credentials", nil)
	req.Header.Set("Authorization", "Bearer")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// "Bearer" with no space splits to 1 part, so bad format
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAdminAuth_TimingResistance(t *testing.T) {
	// Verify that a near-match (same length, one char different) still fails.
	// This doesn't truly test timing, but ensures constant-time compare is wired up.
	env.AdminToken = "abcdefghijklmnopqrstuvwxyz123456"

	handler := AdminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	nearMatch := "abcdefghijklmnopqrstuvwxyz123457" // last char differs
	req := httptest.NewRequest(http.MethodGet, "/admin/credentials", nil)
	req.Header.Set("Authorization", "Bearer "+nearMatch)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestAdminAuth_ResponseFormat(t *testing.T) {
	env.AdminToken = "test-admin-token-12345"

	handler := AdminAuth(http.HandlerFunc(okHandler))

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantJSON   bool
	}{
		{"missing header", "", http.StatusUnauthorized, true},
		{"bad format", "Token abc", http.StatusUnauthorized, true},
		{"wrong token", "Bearer wrong", http.StatusForbidden, true},
		{"valid token", "Bearer test-admin-token-12345", http.StatusOK, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin/credentials", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.wantJSON {
				ct := rr.Header().Get("Content-Type")
				if !strings.Contains(ct, "application/json") {
					t.Fatalf("expected JSON content-type, got %s", ct)
				}
				var errResp responder.JSONError
				if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
					t.Fatalf("response body is not valid JSON: %v", err)
				}
				if errResp.Success {
					t.Fatal("expected success=false in error response")
				}
			}
		})
	}
}
