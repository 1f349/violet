package target

import (
	"net/http"
)

type Route struct {
	Pre  bool
	Host string
	Port int
	Path string
	Abs  bool
}

func (r Route) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// pass
}
