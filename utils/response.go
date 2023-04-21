package utils

import (
	"fmt"
	"net/http"
)

func RespondHttpStatus(rw http.ResponseWriter, status int) {
	http.Error(rw, fmt.Sprintf("%d %s\n", status, http.StatusText(status)), status)
}
