package servers

import (
	"database/sql"
	"fmt"
	"github.com/MrMelon54/violet/domains"
	"github.com/MrMelon54/violet/utils"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"net/url"
	"time"
)

// NewHttpServer creates and runs a http server containing the public http
// endpoints for the reverse proxy.
//
// `/.well-known/acme-challenge/{token}` is used for outputting answers for
// acme challenges, this is used for Lets Encrypt HTTP verification.
func NewHttpServer(listen string, httpsPort int, domainCheck *domains.Domains, db *sql.DB) *http.Server {
	r := httprouter.New()
	var secureExtend string
	if httpsPort != 443 {
		secureExtend = fmt.Sprintf(":%d", httpsPort)
	}

	// Endpoint for acme challenge outputs
	r.GET("/.well-known/acme-challenge/{key}", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		if h, ok := utils.GetDomainWithoutPort(req.Host); ok {
			// check if the host is valid
			if !domainCheck.IsValid(req.Host) {
				http.Error(rw, fmt.Sprintf("%d %s\n", 420, "Invalid host"), 420)
				return
			}

			// check if the key is valid
			key := params.ByName("key")
			if key == "" {
				rw.WriteHeader(http.StatusNotFound)
				return
			}

			// prepare for executing query
			prepare, err := db.Prepare("select value from acme_challenges limit 1 where domain = ? and key = ?")
			if err != nil {
				utils.RespondHttpStatus(rw, http.StatusInternalServerError)
				return
			}

			// query the row and extract the value
			row := prepare.QueryRow(h, key)
			var value string
			err = row.Scan(&value)
			if err != nil {
				utils.RespondHttpStatus(rw, http.StatusInternalServerError)
				return
			}

			// output response
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(value))
		}
		rw.WriteHeader(http.StatusNotFound)
	})

	// All other paths lead here and are forwarded to HTTPS
	r.NotFound = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if h, ok := utils.GetDomainWithoutPort(req.Host); ok {
			u := &url.URL{
				Scheme:   "https",
				Host:     h + secureExtend,
				Path:     req.URL.Path,
				RawPath:  req.URL.RawPath,
				RawQuery: req.URL.RawQuery,
			}
			utils.FastRedirect(rw, req, u.String(), http.StatusPermanentRedirect)
		}
	})

	// Create and run http server
	s := &http.Server{
		Addr:              listen,
		Handler:           r,
		ReadTimeout:       time.Minute,
		ReadHeaderTimeout: time.Minute,
		WriteTimeout:      time.Minute,
		IdleTimeout:       time.Minute,
		MaxHeaderBytes:    2500,
	}
	log.Printf("[HTTP] Starting HTTP server on: '%s'\n", s.Addr)
	go utils.RunBackgroundHttp("HTTP", s)
	return s
}
