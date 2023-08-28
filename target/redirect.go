package target

import (
	"fmt"
	"github.com/1f349/violet/utils"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// Redirect is a target used by the router to manage redirecting the request
// using the specified configuration.
type Redirect struct {
	Src   string `json:"src"`   // request source
	Dst   string `json:"dst"`   // redirect destination
	Flags Flags  `json:"flags"` // extra flags
	Code  int    `json:"code"`  // status code used to redirect
}

type RedirectWithActive struct {
	Redirect
	Active bool `json:"active"`
}

func (r Redirect) HasFlag(flag Flags) bool {
	return r.Flags&flag != 0
}

// ServeHTTP responds with the redirect to the response writer provided.
func (r Redirect) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// default to redirecting with StatusFound if code is not set
	code := r.Code
	if r.Code == 0 {
		code = http.StatusFound
	}

	// split the host and path
	host, p := utils.SplitHostPath(r.Dst)

	// if not Abs then join with the ending of the current path
	if !r.Flags.HasFlag(FlagAbs) {
		p = path.Join(p, req.URL.Path)

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
		Host:   host,
		Path:   p,
	}

	// use fast redirect for speed
	utils.FastRedirect(rw, req, u.String(), code)
}

// String outputs a debug string for the redirect.
func (r Redirect) String() string {
	return fmt.Sprintf("%#v", r)
}
