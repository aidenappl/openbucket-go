package responder

import (
	"encoding/xml"
	"net/http"
)

type ErrorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	RequestId string   `xml:"RequestId"`
	HostId    string   `xml:"HostId"`
}

func SendXML(w http.ResponseWriter, statusCode int, code, message, requestId, hostId string) {

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(statusCode)

	errorResp := ErrorResponse{
		Code:      code,
		Message:   message,
		RequestId: requestId,
		HostId:    hostId,
	}

	xmlData, err := xml.MarshalIndent(errorResp, "", "  ")
	if err != nil {

		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Write(xmlData)
}

func SendAccessDeniedXML(w http.ResponseWriter, requestID *string, hostID *string) {
	if requestID == nil {
		requestID = new(string)
	}
	if hostID == nil {
		hostID = new(string)
	}
	SendXML(w, http.StatusForbidden, "AccessDenied", "Access Denied", *requestID, *hostID)
}
