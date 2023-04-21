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

func GetBearer(req *http.Request) string {
	a := req.Header.Get("Authorization")
	if t, ok := strings.CutPrefix(a, "Bearer "); ok {
		return t
	}
	return ""
}
