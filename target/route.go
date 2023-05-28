package target

import (
	"fmt"
	"github.com/MrMelon54/violet/proxy"
	"github.com/MrMelon54/violet/utils"
	"github.com/rs/cors"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// serveApiCors outputs the cors headers to make APIs work.
var serveApiCors = cors.New(cors.Options{
	AllowedOrigins: []string{"*"}, // allow all origins for api requests
	AllowedHeaders: []string{"Content-Type", "Authorization"},
	AllowedMethods: []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
	},
	AllowCredentials: true,
})

// Route is a target used by the router to manage forwarding traffic to an
// internal server using the specified configuration.
type Route struct {
	Pre         bool         // if the path has had a prefix removed
	Host        string       // target host
	Port        int          // target port
	Path        string       // target path (possibly a prefix or absolute)
	Abs         bool         // if the path is a prefix or absolute
	Cors        bool         // add CORS headers
	SecureMode  bool         // use HTTPS internally
	ForwardHost bool         // forward host header internally
	ForwardAddr bool         // forward remote address
	IgnoreCert  bool         // ignore self-cert
	Headers     http.Header  // extra headers
	Proxy       http.Handler // reverse proxy handler
}

// IsIgnoreCert returns true if IgnoreCert is enabled.
func (r Route) IsIgnoreCert() bool { return r.IgnoreCert }

// UpdateHeaders takes an existing set of headers and overwrites them with the
// extra headers.
func (r Route) UpdateHeaders(header http.Header) {
	for k, v := range r.Headers {
		header[k] = v
	}
}

// FullHost outputs a host:port combo or just the host if the port is 0.
func (r Route) FullHost() string {
	if r.Port == 0 {
		return r.Host
	}
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// ServeHTTP responds with the data proxied from the internal server to the
// response writer provided.
func (r Route) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if r.Cors {
		// wraps with CORS handler
		serveApiCors.Handler(http.HandlerFunc(r.internalServeHTTP)).ServeHTTP(rw, req)
	} else {
		r.internalServeHTTP(rw, req)
	}
}

// internalServeHTTP is an internal method which handles configuring the request
// for the reverse proxy handler.
func (r Route) internalServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// set the scheme and port using defaults if the port is 0
	scheme := "http"
	if r.SecureMode {
		scheme = "https"
		if r.Port == 0 {
			r.Port = 443
		}
	} else {
		if r.Port == 0 {
			r.Port = 80
		}
	}

	// if not Abs then join with the ending of the current path
	p := r.Path
	if !r.Abs {
		p = path.Join(r.Path, req.URL.Path)

		// replace the trailing slash that path.Join() strips off
		if strings.HasSuffix(req.URL.Path, "/") {
			p += "/"
		}
	}

	// fix empty path
	if p == "" {
		p = "/"
	}

	// create a new URL
	u := &url.URL{
		Scheme:   scheme,
		Host:     r.FullHost(),
		Path:     p,
		RawQuery: req.URL.RawQuery,
	}

	// close the incoming body after use
	defer req.Body.Close()

	// create the internal request
	req2, err := http.NewRequest(req.Method, u.String(), req.Body)
	if err != nil {
		log.Printf("[ServeRoute::ServeHTTP()] Error generating new request: %s\n", err)
		utils.RespondHttpStatus(rw, http.StatusBadGateway)
		return
	}

	// loops over the incoming request headers
	for k, v := range req.Header {
		// ignore host header
		if k == "Host" {
			continue
		}
		// copy header into the internal request
		req2.Header[k] = v
	}

	// if extra route headers are set
	if r.Headers != nil {
		// loop over headers
		for k, v := range r.Headers {
			// copy header into the internal request
			req2.Header[k] = v
		}
	}

	// if forward host is enabled then send the host
	if r.ForwardHost {
		req2.Host = req.Host
	}
	if r.ForwardAddr {
		req2.Header.Add("X-Forwarded-For", req.RemoteAddr)
	}

	// serve request with reverse proxy
	r.Proxy.ServeHTTP(rw, proxy.SetReverseProxyHost(req2, r))
}

// String outputs a debug string for the route.
func (r Route) String() string {
	return fmt.Sprintf("%#v", r)
}
