package utils

import (
	"net/http"
)

var (
	a1 = []byte("<a href=\"")
	a2 = []byte("\">")
	a3 = []byte("</a>.\n")
)

func FastRedirect(rw http.ResponseWriter, req *http.Request, url string, code int) {
	rw.Header().Add("Location", url)
	rw.WriteHeader(code)
	if req.Method == http.MethodGet {
		_, _ = rw.Write([]byte(http.StatusText(code)))
	}
}
