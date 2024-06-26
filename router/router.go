package router

import (
	"fmt"
	"github.com/1f349/violet/proxy"
	"github.com/1f349/violet/target"
	"github.com/1f349/violet/utils"
	"github.com/mrmelon54/trie"
	"net/http"
	"strings"
)

type Router struct {
	route    map[string]*trie.Trie[target.Route]
	redirect map[string]*trie.Trie[target.Redirect]
	notFound http.Handler
	proxy    *proxy.HybridTransport
}

func New(proxy *proxy.HybridTransport) *Router {
	return &Router{
		route:    make(map[string]*trie.Trie[target.Route]),
		redirect: make(map[string]*trie.Trie[target.Redirect]),
		notFound: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = fmt.Fprintf(rw, "%d %s\n", http.StatusNotFound, http.StatusText(http.StatusNotFound))
		}),
		proxy: proxy,
	}
}

func (r *Router) hostRoute(host string) *trie.Trie[target.Route] {
	h := r.route[host]
	if h == nil {
		h = &trie.Trie[target.Route]{}
		r.route[host] = h
	}
	return h
}

func (r *Router) hostRedirect(host string) *trie.Trie[target.Redirect] {
	h := r.redirect[host]
	if h == nil {
		h = &trie.Trie[target.Redirect]{}
		r.redirect[host] = h
	}
	return h
}

func (r *Router) AddRoute(t target.Route) {
	t.Proxy = r.proxy
	host, path := utils.SplitHostPath(t.Src)
	r.hostRoute(host).PutString(path, t)
}

func (r *Router) AddRedirect(t target.Redirect) {
	host, path := utils.SplitHostPath(t.Src)
	r.hostRedirect(host).PutString(path, t)
}

func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "" {
		req.URL.Path = "/"
	}

	host, _, _ := utils.SplitDomainPort(req.Host, 0)
	if r.serveRedirectHTTP(rw, req, host) {
		return
	}
	if r.serveRouteHTTP(rw, req, host) {
		return
	}

	parentHostDot := strings.IndexByte(host, '.')
	if parentHostDot == -1 {
		r.notFound.ServeHTTP(rw, req)
		return
	}

	wildcardHost := "*" + host[parentHostDot:]

	if r.serveRedirectHTTP(rw, req, wildcardHost) {
		return
	}
	if r.serveRouteHTTP(rw, req, wildcardHost) {
		return
	}

	utils.RespondVioletError(rw, http.StatusTeapot, "No route")
}

func (r *Router) serveRouteHTTP(rw http.ResponseWriter, req *http.Request, host string) bool {
	h := r.route[host]
	return getServeData(rw, req, h)
}

func (r *Router) serveRedirectHTTP(rw http.ResponseWriter, req *http.Request, host string) bool {
	h := r.redirect[host]
	return getServeData(rw, req, h)
}

type serveDataInterface interface {
	HasFlag(flag target.Flags) bool
	ServeHTTP(rw http.ResponseWriter, req *http.Request)
}

func getServeData[T serveDataInterface](rw http.ResponseWriter, req *http.Request, h *trie.Trie[T]) bool {
	if h == nil {
		return false
	}
	pairs := h.GetAllKeyValues([]byte(req.URL.Path))
	for i := len(pairs) - 1; i >= 0; i-- {
		if pairs[i].Value.HasFlag(target.FlagPre) || pairs[i].Key == req.URL.Path {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, pairs[i].Key)
			pairs[i].Value.ServeHTTP(rw, req)
			return true
		}
	}
	return false
}
