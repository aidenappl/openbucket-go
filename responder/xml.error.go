package responder

import (
	"encoding/xml"
	"log"
	"net/http"
)

type ErrorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	RequestId string   `xml:"RequestId"`
	HostId    string   `xml:"HostId"`
}

func SendXMLError(w http.ResponseWriter, statusCode int, code, message, requestId, hostId string) {

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(statusCode)

	log.Println("Error:", code, message, "RequestId:", requestId, "HostId:", hostId)

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
	SendXMLError(w, http.StatusForbidden, "AccessDenied", "Access Denied", *requestID, *hostID)
}
