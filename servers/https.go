package servers

import (
	"crypto/tls"
	"fmt"
	"github.com/1f349/violet/favicons"
	"github.com/1f349/violet/servers/conf"
	"github.com/1f349/violet/utils"
	"github.com/sethvargo/go-limiter/httplimit"
	"github.com/sethvargo/go-limiter/memorystore"
	"log"
	"net/http"
	"path"
	"runtime"
	"time"
)

// NewHttpsServer creates and runs a http server containing the public https
// endpoints for the reverse proxy.
func NewHttpsServer(conf *conf.Conf) *http.Server {
	r := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		log.Printf("[Debug] Request: %s - '%s' - '%s' - '%s' - len: %d - thread: %d\n", req.Method, req.URL.String(), req.RemoteAddr, req.Host, req.ContentLength, runtime.NumGoroutine())
		conf.Router.ServeHTTP(rw, req)
	})
	favMiddleware := setupFaviconMiddleware(conf.Favicons, r)
	rateLimiter := setupRateLimiter(conf.RateLimit, favMiddleware)

	return &http.Server{
		Addr:    conf.HttpsListen,
		Handler: rateLimiter,
		TLSConfig: &tls.Config{GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			// error out on invalid domains
			if !conf.Domains.IsValid(info.ServerName) {
				return nil, fmt.Errorf("invalid hostname used: '%s'", info.ServerName)
			}

			// find a certificate
			cert := conf.Certs.GetCertForDomain(info.ServerName)
			if cert == nil {
				return nil, fmt.Errorf("failed to find certificate for: '%s'", info.ServerName)
			}

			// time to return
			return cert, nil
		}},
		ReadTimeout:       150 * time.Second,
		ReadHeaderTimeout: 150 * time.Second,
		WriteTimeout:      150 * time.Second,
		IdleTimeout:       150 * time.Second,
		MaxHeaderBytes:    4096000,
	}
}

// setupRateLimiter is an internal function to create a middleware to manage
// rate limits.
func setupRateLimiter(rateLimit uint64, next http.Handler) http.Handler {
	// create memory store
	store, err := memorystore.New(&memorystore.Config{
		Tokens:   rateLimit,
		Interval: time.Minute,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// create a middleware using ips as the key for rate limits
	middleware, err := httplimit.NewMiddleware(store, httplimit.IPKeyFunc())
	if err != nil {
		log.Fatalln(err)
	}
	return middleware.Handle(next)
}

func setupFaviconMiddleware(fav *favicons.Favicons, next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-Violet-Loop-Detect") == "1" {
			rw.WriteHeader(http.StatusLoopDetected)
			_, _ = rw.Write([]byte("Detected a routing loop\n"))
			return
		}
		if req.Header.Get("X-Violet-Raw-Favicon") != "1" {
			switch req.URL.Path {
			case "/favicon.svg", "/favicon.png", "/favicon.ico":
				icons := fav.GetIcons(req.Host)
				if icons == nil {
					break
				}
				raw, contentType, err := icons.ProduceForExt(path.Ext(req.URL.Path))
				if err != nil {
					utils.RespondVioletError(rw, http.StatusTeapot, "No icon available")
					return
				}
				rw.Header().Set("Content-Type", contentType)
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(raw)
				return
			}
		}
		next.ServeHTTP(rw, req)
	})
}
