package responder

import (
	"encoding/json"
	"net/http"
)

type JSONResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type JSONError struct {
	Success      bool   `json:"success"`
	Error        string `json:"error"`
	ErrorMessage string `json:"error_message"`
	ErrorCode    int    `json:"error_code"`
}

func SendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := JSONResponse{
		Success: true,
		Message: "OK",
		Data:    data,
	}

	json.NewEncoder(w).Encode(resp)
}

func SendJSONError(w http.ResponseWriter, statusCode int, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := JSONError{
		Success:      false,
		Error:        http.StatusText(statusCode),
		ErrorMessage: errMsg,
		ErrorCode:    statusCode,
	}

	json.NewEncoder(w).Encode(resp)
}
