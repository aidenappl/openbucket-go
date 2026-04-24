package responder

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendJSON(t *testing.T) {
	rr := httptest.NewRecorder()

	data := map[string]string{"key": "value"}
	SendJSON(rr, http.StatusOK, data)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}

	var resp JSONResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.Message != "OK" {
		t.Fatalf("expected message=OK, got %s", resp.Message)
	}
}

func TestSendJSON_NilData(t *testing.T) {
	rr := httptest.NewRecorder()
	SendJSON(rr, http.StatusOK, nil)

	var resp JSONResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}
}

func TestSendJSONError(t *testing.T) {
	rr := httptest.NewRecorder()
	SendJSONError(rr, http.StatusBadRequest, "something went wrong")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	var resp JSONError
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Success {
		t.Fatal("expected success=false")
	}
	if resp.ErrorMessage != "something went wrong" {
		t.Fatalf("expected error_message='something went wrong', got %s", resp.ErrorMessage)
	}
	if resp.ErrorCode != 400 {
		t.Fatalf("expected error_code=400, got %d", resp.ErrorCode)
	}
	if resp.Error != "Bad Request" {
		t.Fatalf("expected error='Bad Request', got %s", resp.Error)
	}
}

func TestSendJSON_CreatedStatus(t *testing.T) {
	rr := httptest.NewRecorder()
	SendJSON(rr, http.StatusCreated, map[string]int{"id": 42})

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestSendJSONError_AllStatusCodes(t *testing.T) {
	tests := []struct {
		status   int
		wantText string
	}{
		{http.StatusUnauthorized, "Unauthorized"},
		{http.StatusForbidden, "Forbidden"},
		{http.StatusNotFound, "Not Found"},
		{http.StatusInternalServerError, "Internal Server Error"},
		{http.StatusBadGateway, "Bad Gateway"},
	}

	for _, tt := range tests {
		t.Run(tt.wantText, func(t *testing.T) {
			rr := httptest.NewRecorder()
			SendJSONError(rr, tt.status, "test error")

			if rr.Code != tt.status {
				t.Fatalf("expected %d, got %d", tt.status, rr.Code)
			}

			var resp JSONError
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode: %v", err)
			}
			if resp.Error != tt.wantText {
				t.Fatalf("expected error=%q, got %q", tt.wantText, resp.Error)
			}
			if resp.ErrorCode != tt.status {
				t.Fatalf("expected error_code=%d, got %d", tt.status, resp.ErrorCode)
			}
		})
	}
}

func TestSendJSON_DataPreserved(t *testing.T) {
	rr := httptest.NewRecorder()

	type item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	data := []item{{ID: 1, Name: "alpha"}, {ID: 2, Name: "beta"}}
	SendJSON(rr, http.StatusOK, data)

	var resp struct {
		Success bool   `json:"success"`
		Data    []item `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Data))
	}
	if resp.Data[0].Name != "alpha" || resp.Data[1].Name != "beta" {
		t.Fatalf("data mismatch: %+v", resp.Data)
	}
}
