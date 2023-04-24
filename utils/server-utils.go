package utils

import (
	"log"
	"net/http"
	"strings"
)

func logHttpServerError(prefix string, err error) {
	if err != nil {
		if err == http.ErrServerClosed {
			log.Printf("[%s] The http server shutdown successfully\n", prefix)
		} else {
			log.Printf("[%s] Error trying to host the http server: %s\n", prefix, err.Error())
		}
	}
}

func RunBackgroundHttp(prefix string, s *http.Server) {
	logHttpServerError(prefix, s.ListenAndServe())
}

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
