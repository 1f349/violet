package utils

import (
	"log"
	"net/http"
	"strings"
)

// logHttpServerError is the internal function powering the logging in
// RunBackgroundHttp and RunBackgroundHttps.
func logHttpServerError(prefix string, err error) {
	if err != nil {
		if err == http.ErrServerClosed {
			log.Printf("[%s] The http server shutdown successfully\n", prefix)
		} else {
			log.Printf("[%s] Error trying to host the http server: %s\n", prefix, err.Error())
		}
	}
}

// RunBackgroundHttp runs a http server and logs when the server closes or
// errors.
func RunBackgroundHttp(prefix string, s *http.Server) {
	logHttpServerError(prefix, s.ListenAndServe())
}

// RunBackgroundHttps runs a http server with TLS encryption and logs when the
// server closes or errors.
func RunBackgroundHttps(prefix string, s *http.Server) {
	logHttpServerError(prefix, s.ListenAndServeTLS("", ""))
}

// GetBearer returns the bearer from the Authorization header or an empty string
// if the authorization is empty or doesn't start with Bearer.
func GetBearer(req *http.Request) string {
	a := req.Header.Get("Authorization")
	if t, ok := strings.CutPrefix(a, "Bearer "); ok {
		return t
	}
	return ""
}
