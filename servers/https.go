package servers

import (
	"crypto/tls"
	"fmt"
	"github.com/MrMelon54/violet/favicons"
	"github.com/MrMelon54/violet/utils"
	"github.com/sethvargo/go-limiter/httplimit"
	"github.com/sethvargo/go-limiter/memorystore"
	"log"
	"net"
	"net/http"
	"time"
)

// NewHttpsServer creates and runs a http server containing the public https
// endpoints for the reverse proxy.
func NewHttpsServer(conf *Conf) *http.Server {
	return &http.Server{
		Addr:    conf.HttpsListen,
		Handler: setupRateLimiter(conf.RateLimit, setupFaviconMiddleware(conf.Favicons, conf.Router)),
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
		ConnState: func(conn net.Conn, state http.ConnState) {
			fmt.Printf("[HTTPS] %s => %s: %s\n", conn.LocalAddr(), conn.RemoteAddr(), state.String())
		},
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
		if req.Header.Get("X-Violet-Raw-Favicon") != "1" {
			switch req.URL.Path {
			case "/favicon.svg":
				icons := fav.GetIcons(req.Host)
				raw, err := icons.ProduceSvg()
				if err != nil {
					utils.RespondVioletError(rw, http.StatusTeapot, "No SVG icon available")
					return
				}
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(raw)
				return
			case "/favicon.png":
				icons := fav.GetIcons(req.Host)
				raw, err := icons.ProducePng()
				if err != nil {
					utils.RespondVioletError(rw, http.StatusTeapot, "No PNG icon available")
					return
				}
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(raw)
				return
			case "/favicon.ico":
				icons := fav.GetIcons(req.Host)
				raw, err := icons.ProduceIco()
				if err != nil {
					utils.RespondVioletError(rw, http.StatusTeapot, "No ICO icon available")
					return
				}
				rw.WriteHeader(http.StatusOK)
				_, _ = rw.Write(raw)
				return
			}
		}
		next.ServeHTTP(rw, req)
	})
}
