package servers

import (
	"fmt"
	"github.com/1f349/violet/servers/conf"
	"github.com/1f349/violet/utils"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"net/url"
	"time"
)

// NewHttpServer creates and runs a http server containing the public http
// endpoints for the reverse proxy.
//
// `/.well-known/acme-challenge/{token}` is used for outputting answers for
// acme challenges, this is used for Let's Encrypt HTTP verification.
func NewHttpServer(conf *conf.Conf) *http.Server {
	r := httprouter.New()
	var secureExtend string
	_, httpsPort, ok := utils.SplitDomainPort(conf.HttpsListen, 443)
	if !ok {
		httpsPort = 443
	}
	if httpsPort != 443 {
		secureExtend = fmt.Sprintf(":%d", httpsPort)
	}

	// Endpoint for acme challenge outputs
	r.GET("/.well-known/acme-challenge/:key", func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		h := utils.GetDomainWithoutPort(req.Host)

		// check if the host is valid
		if !conf.Domains.IsValid(req.Host) {
			utils.RespondVioletError(rw, http.StatusBadRequest, "Invalid host")
			return
		}

		// check if the key is valid
		value := conf.Acme.Get(h, params.ByName("key"))
		if value == "" {
			rw.WriteHeader(http.StatusNotFound)
			return
		}

		// output response
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte(value))
	})

	// All other paths lead here and are forwarded to HTTPS
	r.NotFound = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		h := utils.GetDomainWithoutPort(req.Host)
		u := &url.URL{
			Scheme:   "https",
			Host:     h + secureExtend,
			Path:     req.URL.Path,
			RawPath:  req.URL.RawPath,
			RawQuery: req.URL.RawQuery,
		}
		utils.FastRedirect(rw, req, u.String(), http.StatusPermanentRedirect)
	})

	// Create and run http server
	return &http.Server{
		Addr:              conf.HttpListen,
		Handler:           r,
		ReadTimeout:       time.Minute,
		ReadHeaderTimeout: time.Minute,
		WriteTimeout:      time.Minute,
		IdleTimeout:       time.Minute,
		MaxHeaderBytes:    2500,
	}
}
