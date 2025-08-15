package tools

import (
	"net/http"
	"strings"
)

// HeaderExists returns (found, headerName, value) for the first matching header.
// found is true if any given header exists in the request and has a non-empty value.
func HeaderExists(r *http.Request, headers ...string) (bool, string, string) {
	if r == nil || len(headers) == 0 {
		return false, "", ""
	}

	for _, h := range headers {
		if h == "" {
			continue
		}
		if v := r.Header.Get(h); strings.TrimSpace(v) != "" {
			return true, h, v
		}
	}
	return false, "", ""
}
