package router

import (
	"github.com/1f349/violet/proxy"
	"github.com/1f349/violet/target"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"
)

type routeTestBase struct {
	path  string
	dst   target.Route
	tests map[string]string
}

type redirectTestBase struct {
	path  string
	dst   target.Redirect
	tests map[string]string
}

type mss map[string]string

var (
	routeTests = []routeTestBase{
		{"/", target.Route{}, mss{
			"/":      "/",
			"/hello": "",
		}},
		{"/", target.Route{Dst: "world"}, mss{
			"/":      "/world",
			"/hello": "",
		}},
		{"/", target.Route{Flags: target.FlagAbs}, mss{
			"/":      "/",
			"/hello": "",
		}},
		{"/", target.Route{Flags: target.FlagAbs, Dst: "world"}, mss{
			"/":      "/world",
			"/hello": "",
		}},
		{"/", target.Route{Flags: target.FlagPre}, mss{
			"/":      "/",
			"/hello": "/hello",
		}},
		{"/", target.Route{Flags: target.FlagPre, Dst: "world"}, mss{
			"/":      "/world",
			"/hello": "/world/hello",
		}},
		{"/", target.Route{Flags: target.FlagPre | target.FlagAbs}, mss{
			"/":      "/",
			"/hello": "/",
		}},
		{"/", target.Route{Flags: target.FlagPre | target.FlagAbs, Dst: "world"}, mss{
			"/":      "/world",
			"/hello": "/world",
		}},
		{"/hello", target.Route{}, mss{
			"/":         "",
			"/hello":    "/",
			"/hello/hi": "",
		}},
		{"/hello", target.Route{Dst: "world"}, mss{
			"/":         "",
			"/hello":    "/world",
			"/hello/hi": "",
		}},
		{"/hello", target.Route{Flags: target.FlagAbs}, mss{
			"/":         "",
			"/hello":    "/",
			"/hello/hi": "",
		}},
		{"/hello", target.Route{Flags: target.FlagAbs, Dst: "world"}, mss{
			"/":         "",
			"/hello":    "/world",
			"/hello/hi": "",
		}},
		{"/hello", target.Route{Flags: target.FlagPre}, mss{
			"/":         "",
			"/hello":    "/",
			"/hello/hi": "/hi",
		}},
		{"/hello", target.Route{Flags: target.FlagPre, Dst: "world"}, mss{
			"/":         "",
			"/hello":    "/world",
			"/hello/hi": "/world/hi",
		}},
		{"/hello", target.Route{Flags: target.FlagPre | target.FlagAbs}, mss{
			"/":         "",
			"/hello":    "/",
			"/hello/hi": "/",
		}},
		{"/hello", target.Route{Flags: target.FlagPre | target.FlagAbs, Dst: "world"}, mss{
			"/":         "",
			"/hello":    "/world",
			"/hello/hi": "/world",
		}},
	}
	redirectTests = []redirectTestBase{
		{"/", target.Redirect{}, mss{
			"/":      "/",
			"/hello": "",
		}},
		{"/", target.Redirect{Dst: "world"}, mss{
			"/":      "/world",
			"/hello": "",
		}},
		{"/", target.Redirect{Flags: target.FlagAbs}, mss{
			"/":      "/",
			"/hello": "",
		}},
		{"/", target.Redirect{Flags: target.FlagAbs, Dst: "world"}, mss{
			"/":      "/world",
			"/hello": "",
		}},
		{"/", target.Redirect{Flags: target.FlagPre}, mss{
			"/":      "/",
			"/hello": "/hello",
		}},
		{"/", target.Redirect{Flags: target.FlagPre, Dst: "world"}, mss{
			"/":      "/world",
			"/hello": "/world/hello",
		}},
		{"/", target.Redirect{Flags: target.FlagPre | target.FlagAbs}, mss{
			"/":      "/",
			"/hello": "/",
		}},
		{"/", target.Redirect{Flags: target.FlagPre | target.FlagAbs, Dst: "world"}, mss{
			"/":      "/world",
			"/hello": "/world",
		}},
		{"/hello", target.Redirect{}, mss{
			"/":         "",
			"/hello":    "/",
			"/hello/hi": "",
		}},
		{"/hello", target.Redirect{Dst: "world"}, mss{
			"/":         "",
			"/hello":    "/world",
			"/hello/hi": "",
		}},
		{"/hello", target.Redirect{Flags: target.FlagAbs}, mss{
			"/":         "",
			"/hello":    "/",
			"/hello/hi": "",
		}},
		{"/hello", target.Redirect{Flags: target.FlagAbs, Dst: "world"}, mss{
			"/":         "",
			"/hello":    "/world",
			"/hello/hi": "",
		}},
		{"/hello", target.Redirect{Flags: target.FlagPre}, mss{
			"/":         "",
			"/hello":    "/",
			"/hello/hi": "/hi",
		}},
		{"/hello", target.Redirect{Flags: target.FlagPre, Dst: "world"}, mss{
			"/":         "",
			"/hello":    "/world",
			"/hello/hi": "/world/hi",
		}},
		{"/hello", target.Redirect{Flags: target.FlagPre | target.FlagAbs}, mss{
			"/":         "",
			"/hello":    "/",
			"/hello/hi": "/",
		}},
		{"/hello", target.Redirect{Flags: target.FlagPre | target.FlagAbs, Dst: "world"}, mss{
			"/":         "",
			"/hello":    "/world",
			"/hello/hi": "/world",
		}},
	}
)

func TestRouter_AddRoute(t *testing.T) {
	transSecure := &fakeTransport{}
	transInsecure := &fakeTransport{}

	for _, i := range routeTests {
		r := New(proxy.NewHybridTransportWithCalls(transSecure, transInsecure))
		dst := i.dst
		dst.Dst = path.Join("127.0.0.1:8080", dst.Dst)
		dst.Src = path.Join("example.com", i.path)
		t.Logf("Running tests for %#v\n", dst)
		r.AddRoute(dst)
		for k, v := range i.tests {
			u1 := &url.URL{Scheme: "https", Host: "example.com", Path: k}
			req, _ := http.NewRequest(http.MethodGet, u1.String(), nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if v == "" {
				if transSecure.req != nil {
					t.Logf("Test URL: %#v\n", req.URL)
					t.Log(r.route["example.com"].String())
					t.Fatalf("%s => %s\n", k, v)
				}
			} else {
				if transSecure.req == nil {
					t.Logf("Test URL: %#v\n", req.URL)
					t.Log(r.route["example.com"].String())
					t.Fatalf("\nexpected %s => %s\n     got %s => %s\n", k, v, k, "")
				}
				if v != transSecure.req.URL.Path {
					t.Logf("Test URL: %#v\n", req.URL)
					t.Log(r.route["example.com"].String())
					t.Fatalf("\nexpected %s => %s\n     got %s => %s\n", k, v, k, transSecure.req.URL.Path)
				}
				transSecure.req = nil
			}
		}
	}
}

func TestRouter_AddRedirect(t *testing.T) {
	for _, i := range redirectTests {
		r := New(nil)
		dst := i.dst
		dst.Dst = path.Join("example.com", dst.Dst)
		dst.Code = http.StatusFound
		dst.Src = path.Join("www.example.com", i.path)
		t.Logf("Running tests for %#v\n", dst)
		r.AddRedirect(dst)
		for k, v := range i.tests {
			u1 := &url.URL{Scheme: "https", Host: "example.com", Path: v}
			if v == "" {
				u1 = nil
			}
			u2 := &url.URL{Scheme: "https", Host: "www.example.com", Path: k}
			assertHttpRedirect(t, r, http.StatusFound, outputUrl(u1), http.MethodGet, outputUrl(u2))
		}
	}
}

func assertHttpRedirect(t *testing.T, r *Router, code int, target, method, start string) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(method, start, nil)
	r.ServeHTTP(res, req)
	l := res.Header().Get("Location")
	if target == "" {
		if code == res.Code || "" != l {
			t.Logf("Test URL: %#v\n", req.URL)
			t.Log(r.redirect["www.example.com"].String())
			t.Fatalf("%s => %s\n", start, target)
		}
	} else {
		if code != res.Code || target != l {
			t.Logf("Test URL: %#v\n", req.URL)
			t.Log(r.redirect["www.example.com"].String())
			t.Fatalf("\nexpected %s => %s\n     got %s => %s\n", start, target, start, l)
		}
	}
}

func outputUrl(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.String()
}

func TestRouter_AddWildcardRoute(t *testing.T) {
	transSecure := &fakeTransport{}
	transInsecure := &fakeTransport{}

	for _, i := range routeTests {
		r := New(proxy.NewHybridTransportWithCalls(transSecure, transInsecure))
		dst := i.dst
		dst.Dst = path.Join("127.0.0.1:8080", dst.Dst)
		dst.Src = path.Join("*.example.com", i.path)
		t.Logf("Running tests for %#v\n", dst)
		r.AddRoute(dst)
		for k, v := range i.tests {
			u1 := &url.URL{Scheme: "https", Host: "test.example.com", Path: k}
			req, _ := http.NewRequest(http.MethodGet, u1.String(), nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if v == "" {
				if transSecure.req != nil {
					t.Logf("Test URL: %#v\n", req.URL)
					t.Log(r.route["*.example.com"].String())
					t.Fatalf("%s => %s\n", k, v)
				}
			} else {
				if transSecure.req == nil {
					t.Logf("Test URL: %#v\n", req.URL)
					t.Log(r.route["*.example.com"].String())
					t.Fatalf("\nexpected %s => %s\n     got %s => %s\n", k, v, k, "")
				}
				if v != transSecure.req.URL.Path {
					t.Logf("Test URL: %#v\n", req.URL)
					t.Log(r.route["*.example.com"].String())
					t.Fatalf("\nexpected %s => %s\n     got %s => %s\n", k, v, k, transSecure.req.URL.Path)
				}
				transSecure.req = nil
			}
		}
	}
}
