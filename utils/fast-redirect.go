package utils

import (
	"net/http"
)

// FastRedirect adds a location header, status code and if the method is get,
// outputs the status text.
func FastRedirect(rw http.ResponseWriter, req *http.Request, url string, code int) {
	rw.Header().Add("Location", url)
	rw.WriteHeader(code)
	if req.Method == http.MethodGet {
		_, _ = rw.Write([]byte(http.StatusText(code)))
	}
}
