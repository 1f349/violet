package utils

import (
	"fmt"
	"net/http"
)

// RespondHttpStatus outputs the status code and text using http.Error()
func RespondHttpStatus(rw http.ResponseWriter, status int) {
	http.Error(rw, fmt.Sprintf("%d %s\n", status, http.StatusText(status)), status)
}

func RespondVioletError(rw http.ResponseWriter, status int, msg string) {
	rw.Header().Set("X-Violet-Error", msg)
	RespondHttpStatus(rw, status)
}
