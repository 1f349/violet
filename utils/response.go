package utils

import (
	"fmt"
	"net/http"
)

// RespondHttpStatus outputs the status code and text using http.Error()
func RespondHttpStatus(rw http.ResponseWriter, status int) {
	http.Error(rw, fmt.Sprintf("%d %s\n", status, http.StatusText(status)), status)
}
