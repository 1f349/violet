package servers

import (
	"crypto/tls"
	"fmt"
	"github.com/1f349/violet/favicons"
	"github.com/1f349/violet/logger"
	"github.com/1f349/violet/servers/conf"
	"github.com/1f349/violet/servers/metrics"
	"github.com/1f349/violet/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sethvargo/go-limiter/httplimit"
	"github.com/sethvargo/go-limiter/memorystore"
	"net/http"
	"path"
	"runtime"
	"time"
)

// NewHttpsServer creates and runs a http server containing the public https
// endpoints for the reverse proxy.
func NewHttpsServer(conf *conf.Conf, registry *prometheus.Registry) *http.Server {
	r := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		logger.Logger.Debug("Request", "method", req.Method, "url", req.URL, "remote", req.RemoteAddr, "host", req.Host, "length", req.ContentLength, "goroutine", runtime.NumGoroutine())
		conf.Router.ServeHTTP(rw, req)
	})
	favMiddleware := setupFaviconMiddleware(conf.Favicons, r)

	metricsMeta := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		r.ServeHTTP(rw, req)
	})
	if registry != nil {
		metricsMiddleware := metrics.New(registry, nil).WrapHandler("violet-https", favMiddleware)
		metricsMeta = func(rw http.ResponseWriter, req *http.Request) {
			metricsMiddleware.ServeHTTP(rw, metrics.AddHostCtx(req))
		}
	}
	rateLimiter := setupRateLimiter(conf.RateLimit, metricsMeta)
	hsts := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		rateLimiter.ServeHTTP(rw, req)
	})

	return &http.Server{
		Handler: hsts,
		TLSConfig: &tls.Config{
			// Suggested by https://ssl-config.mozilla.org/#server=go&version=1.21.5&config=intermediate
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			},
			GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
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
			},
		},
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
		logger.Logger.Fatal("Failed to initialize memory store", "err", err)
	}

	// create a middleware using ips as the key for rate limits
	middleware, err := httplimit.NewMiddleware(store, httplimit.IPKeyFunc())
	if err != nil {
		logger.Logger.Fatal("Failed to initialize httplimit middleware", "err", err)
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
