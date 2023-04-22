package error_pages

import (
	"fmt"
	"net/http"
	"sync"
)

// ErrorPages stores the custom error pages and is called by the servers to
// output meaningful pages for HTTP error codes
type ErrorPages struct {
	s       *sync.RWMutex
	m       map[int]func(rw http.ResponseWriter)
	generic func(rw http.ResponseWriter, code int)
	dir     string
}

func New(dir string) *ErrorPages {
	return &ErrorPages{
		s: &sync.RWMutex{},
		m: make(map[int]func(rw http.ResponseWriter)),
		generic: func(rw http.ResponseWriter, code int) {
			a := http.StatusText(code)
			if a != "" {
				http.Error(rw, fmt.Sprintf("%d %s\n", code, a), code)
				return
			}
			http.Error(rw, fmt.Sprintf("%d Unknown Error Code\n", code), code)
		},
		dir: dir,
	}
}

func (e *ErrorPages) Compile() {

}

func (e *ErrorPages) ServeError(rw http.ResponseWriter, code int) {
	e.s.RLock()
	defer e.s.RUnlock()
	if p, ok := e.m[code]; ok {
		p(rw)
		return
	}
	e.generic(rw, code)
}
