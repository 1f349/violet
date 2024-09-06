package utils

import (
	"errors"
	"github.com/charmbracelet/log"
	"net"
	"net/http"
	"strings"
)

// logHttpServerError is the internal function powering the logging in
// RunBackgroundHttp and RunBackgroundHttps.
func logHttpServerError(logger *log.Logger, err error) {
	if err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			logger.Info("The http server shutdown successfully")
		} else {
			logger.Info("Error trying to host the http server", "err", err.Error())
		}
	}
}

// RunBackgroundHttp runs a http server and logs when the server closes or
// errors.
func RunBackgroundHttp(logger *log.Logger, s *http.Server, ln net.Listener) {
	logHttpServerError(logger, s.Serve(ln))
}

// RunBackgroundHttps runs a http server with TLS encryption and logs when the
// server closes or errors.
func RunBackgroundHttps(logger *log.Logger, s *http.Server, ln net.Listener) {
	logHttpServerError(logger, s.ServeTLS(ln, "", ""))
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
