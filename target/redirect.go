package target

import (
	"fmt"
	"github.com/MrMelon54/violet/utils"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// Redirect is a target used by the router to manage redirecting the request
// using the specified configuration.
type Redirect struct {
	Pre  bool   // if the path has had a prefix removed
	Host string // target host
	Port int    // target port
	Path string // target path (possibly a prefix or absolute)
	Abs  bool   // if the path is a prefix or absolute
	Code int    // status code used to redirect
}

// FullHost outputs a host:port combo or just the host if the port is 0.
func (r Redirect) FullHost() string {
	if r.Port == 0 {
		return r.Host
	}
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// ServeHTTP responds with the redirect to the response writer provided.
func (r Redirect) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// default to redirecting with StatusFound if code is not set
	code := r.Code
	if r.Code == 0 {
		code = http.StatusFound
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
		Scheme: req.URL.Scheme,
		Host:   r.FullHost(),
		Path:   p,
	}

	// use fast redirect for speed
	utils.FastRedirect(rw, req, u.String(), code)
}

// String outputs a debug string for the redirect.
func (r Redirect) String() string {
	return fmt.Sprintf("%#v", r)
}
