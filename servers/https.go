package servers

import (
	"crypto/tls"
	"fmt"
	"github.com/MrMelon54/violet/router"
	"github.com/MrMelon54/violet/utils"
	"github.com/gorilla/mux"
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
	r := router.New(conf.Proxy)

	s := &http.Server{
		Addr: conf.HttpsListen,
		Handler: setupRateLimiter(300).Middleware(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.Header().Set("Content-Type", "text/html")
			rw.WriteHeader(http.StatusNotImplemented)
			_, _ = rw.Write([]byte("<pre>"))
			_, _ = rw.Write([]byte(fmt.Sprintf("%#v\n", req)))
			_, _ = rw.Write([]byte("</pre>"))
			_ = r
			// TODO: serve from router and proxy
			// r.ServeHTTP(rw, req)
		})),
		DisableGeneralOptionsHandler: false,
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
			fmt.Printf("%s => %s: %s\n", conn.LocalAddr(), conn.RemoteAddr(), state.String())
		},
	}
	log.Printf("[HTTPS] Starting HTTPS server on: '%s'\n", s.Addr)
	go utils.RunBackgroundHttps("HTTPS", s)
	return s
}

// setupRateLimiter is an internal function to create a middleware to manage
// rate limits.
func setupRateLimiter(rateLimit uint64) mux.MiddlewareFunc {
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
	return middleware.Handle
}
