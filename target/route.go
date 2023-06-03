package target

import (
	"context"
	"fmt"
	"github.com/MrMelon54/violet/proxy"
	"github.com/MrMelon54/violet/utils"
	"github.com/rs/cors"
	"golang.org/x/net/http/httpguts"
	"io"
	"log"
	"net"
	"net/http"
	"net/textproto"
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
	Pre         bool                   // if the path has had a prefix removed
	Host        string                 // target host
	Port        int                    // target port
	Path        string                 // target path (possibly a prefix or absolute)
	Abs         bool                   // if the path is a prefix or absolute
	Cors        bool                   // add CORS headers
	SecureMode  bool                   // use HTTPS internally
	ForwardHost bool                   // forward host header internally
	ForwardAddr bool                   // forward remote address
	IgnoreCert  bool                   // ignore self-cert
	Headers     http.Header            // extra headers
	Proxy       *proxy.HybridTransport // reverse proxy handler
}

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
		utils.RespondVioletError(rw, http.StatusBadGateway, "error generating new request")
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

	// adds extra request metadata
	r.internalReverseProxyMeta(rw, req)

	// serve request with reverse proxy
	var resp *http.Response
	if r.IgnoreCert {
		resp, err = r.Proxy.InsecureRoundTrip(req2)
	} else {
		resp, err = r.Proxy.SecureRoundTrip(req2)
	}

	// copy headers and status code
	copyHeader(rw.Header(), resp.Header)
	rw.WriteHeader(resp.StatusCode)

	// copy body
	if resp.Body != nil {
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			// hijack and close upon error
			if h, ok := rw.(http.Hijacker); ok {
				hijack, _, err := h.Hijack()
				if err != nil {
					return
				}
				_ = hijack.Close()
			}
			return
		}
	}
}

// internalReverseProxyMeta is mainly built from code copied from httputil.ReverseProxy,
// due to the highly custom nature of this reverse proxy software we use a copy
// of the code instead of the full httputil implementation to prevent overhead
// from the more generic implementation
func (r Route) internalReverseProxyMeta(rw http.ResponseWriter, req *http.Request) {
	outreq := req.Clone(context.Background())
	if req.ContentLength == 0 {
		outreq.Body = nil // Issue 16036: nil Body for http.Transport retries
	}
	if outreq.Body != nil {
		// Reading from the request body after returning from a handler is not
		// allowed, and the RoundTrip goroutine that reads the Body can outlive
		// this handler. This can lead to a crash if the handler panics (see
		// Issue 46866). Although calling Close doesn't guarantee there isn't
		// any Read in flight after the handle returns, in practice it's safe to
		// read after closing it.
		defer outreq.Body.Close()
	}
	if outreq.Header == nil {
		outreq.Header = make(http.Header) // Issue 33142: historical behavior was to always allocate
	}

	reqUpType := upgradeType(outreq.Header)
	if !asciiIsPrint(reqUpType) {
		utils.RespondVioletError(rw, http.StatusBadRequest, fmt.Sprintf("client tried to switch to invalid protocol %q", reqUpType))
		return
	}
	removeHopByHopHeaders(outreq.Header)

	// Issue 21096: tell backend applications that care about trailer support
	// that we support trailers. (We do, but we don't go out of our way to
	// advertise that unless the incoming client request thought it was worth
	// mentioning.) Note that we look at req.Header, not outreq.Header, since
	// the latter has passed through removeHopByHopHeaders.
	if httpguts.HeaderValuesContainsToken(req.Header["Te"], "trailers") {
		outreq.Header.Set("Te", "trailers")
	}

	// After stripping all the hop-by-hop connection headers above, add back any
	// necessary for protocol upgrades, such as for websockets.
	if reqUpType != "" {
		outreq.Header.Set("Connection", "Upgrade")
		outreq.Header.Set("Upgrade", reqUpType)
	}

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		// If we aren't the first proxy retain prior
		// X-Forwarded-For information as a comma+space
		// separated list and fold multiple headers into one.
		prior, ok := outreq.Header["X-Forwarded-For"]
		omit := ok && prior == nil // Issue 38079: nil now means don't populate the header
		if len(prior) > 0 {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		if !omit {
			outreq.Header.Set("X-Forwarded-For", clientIP)
		}
	}
}

// String outputs a debug string for the route.
func (r Route) String() string {
	return fmt.Sprintf("%#v", r)
}

// copyHeader copies all headers from src to dst
func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// updateType returns the value of upgrade from http.Header
func upgradeType(h http.Header) string {
	if !httpguts.HeaderValuesContainsToken(h["Connection"], "Upgrade") {
		return ""
	}
	return h.Get("Upgrade")
}

// IsPrint returns whether s is ASCII and printable according to
// https://tools.ietf.org/html/rfc20#section-4.2.
func asciiIsPrint(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < ' ' || s[i] > '~' {
			return false
		}
	}
	return true
}

// Hop-by-hop headers. These are removed when sent to the backend.
// As of RFC 7230, hop-by-hop headers are required to appear in the
// Connection header field. These are the headers defined by the
// obsoleted RFC 2616 (section 13.5.1) and are used for backward
// compatibility.
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection", // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // not Trailers per URL above; https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}

// removeHopByHopHeaders removes the hop-by-hop headers defined in hopHeaders
func removeHopByHopHeaders(h http.Header) {
	// RFC 7230, section 6.1: Remove headers listed in the "Connection" header.
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = textproto.TrimString(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
	// RFC 2616, section 13.5.1: Remove a set of known hop-by-hop headers.
	// This behavior is superseded by the RFC 7230 Connection header, but
	// preserve it for backwards compatibility.
	for _, f := range hopHeaders {
		h.Del(f)
	}
}
