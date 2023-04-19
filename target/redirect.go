package target

import (
	"net/http"
	"net/url"
)

type Redirect struct {
	url.URL
	Code int
}

func (r Redirect) Handler() http.Handler {
	return http.RedirectHandler(r.URL.String(), r.Code)
}
