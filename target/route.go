package target

import (
	"bytes"
	"fmt"
	"github.com/MrMelon54/violet/proxy"
	"github.com/MrMelon54/violet/utils"
	"github.com/rs/cors"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
)

var serveApiCors = cors.New(cors.Options{
	AllowedOrigins: []string{"*"},
	AllowedHeaders: []string{"Content-Type", "Authorization"},
	AllowedMethods: []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	},
	AllowCredentials: true,
})

type Route struct {
	Pre         bool
	Host        string
	Port        int
	Path        string
	Abs         bool
	Cors        bool
	SecureMode  bool
	ForwardHost bool
	IgnoreCert  bool
	Headers     http.Header
	Proxy       http.Handler
}

func (r Route) IsIgnoreCert() bool { return r.IgnoreCert }

func (r Route) UpdateHeaders(header http.Header) {
	for k, v := range r.Headers {
		header[k] = v
	}
}

func (r Route) FullHost() string {
	if r.Port == 0 {
		return r.Host
	}
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

func (r Route) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if r.Cors {
		serveApiCors.Handler(http.HandlerFunc(r.internalServeHTTP)).ServeHTTP(rw, req)
	} else {
		r.internalServeHTTP(rw, req)
	}
}

func (r Route) internalServeHTTP(rw http.ResponseWriter, req *http.Request) {
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

	p := r.Path
	if !r.Abs {
		p = path.Join(r.Path, req.URL.Path)
	}

	if p == "" {
		p = "/"
	}

	buf := new(bytes.Buffer)
	if req.Body != nil {
		_, _ = io.Copy(buf, req.Body)
	}

	u := &url.URL{
		Scheme:   scheme,
		Host:     r.FullHost(),
		Path:     p,
		RawQuery: req.URL.RawQuery,
	}
	req2, err := http.NewRequest(req.Method, u.String(), buf)
	if err != nil {
		log.Printf("[ServeRoute::ServeHTTP()] Error generating new request: %s\n", err)
		utils.RespondHttpStatus(rw, http.StatusBadGateway)
		return
	}
	for k, v := range req.Header {
		if k == "Host" {
			continue
		}
		req2.Header[k] = v
	}
	if r.ForwardHost {
		req2.Host = req.Host
	}
	r.Proxy.ServeHTTP(rw, proxy.SetReverseProxyHost(req2, r))
}
