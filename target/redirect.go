package target

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
)

type Redirect struct {
	Pre  bool
	Host string
	Port int
	Path string
	Abs  bool
	Code int
}

func (r Redirect) FullHost() string {
	if r.Port == 0 {
		return r.Host
	}
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

func (r Redirect) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p := r.Path
	if !r.Abs {
		p = path.Join(r.Path, req.URL.Path)
	}
	u := url.URL{
		Scheme: req.URL.Scheme,
		Host:   r.FullHost(),
		Path:   p,
	}
	if u.Path == "/" {
		u.Path = ""
	}
	http.Redirect(rw, req, u.String(), r.Code)
}

func (r Redirect) String() string {
	return fmt.Sprintf("%#v", r)
}
