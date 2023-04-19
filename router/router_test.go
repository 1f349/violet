package router

import (
	"github.com/MrMelon54/violet/target"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type redirectTestBase struct {
	path  string
	dst   target.Redirect
	tests map[string]string
}

type mss map[string]string

var redirectTests = []redirectTestBase{
	{"/", target.Redirect{}, mss{
		"/":      "/",
		"/hello": "",
	}},
	{"/", target.Redirect{Path: "world"}, mss{
		"/":      "/world",
		"/hello": "",
	}},
	{"/", target.Redirect{Abs: true}, mss{
		"/":      "/",
		"/hello": "",
	}},
	{"/", target.Redirect{Abs: true, Path: "world"}, mss{
		"/":      "/world",
		"/hello": "",
	}},
	{"/", target.Redirect{Pre: true}, mss{
		"/":      "/",
		"/hello": "/hello",
	}},
	{"/", target.Redirect{Pre: true, Path: "world"}, mss{
		"/":      "/world",
		"/hello": "/world/hello",
	}},
	{"/", target.Redirect{Pre: true, Abs: true}, mss{
		"/":      "/",
		"/hello": "/",
	}},
	{"/", target.Redirect{Pre: true, Abs: true, Path: "world"}, mss{
		"/":      "/world",
		"/hello": "/world",
	}},
	{"/hello", target.Redirect{}, mss{
		"/":         "",
		"/hello":    "/",
		"/hello/hi": "",
	}},
	{"/hello", target.Redirect{Path: "world"}, mss{
		"/":         "",
		"/hello":    "/world",
		"/hello/hi": "",
	}},
	{"/hello", target.Redirect{Abs: true}, mss{
		"/":         "",
		"/hello":    "/",
		"/hello/hi": "",
	}},
	{"/hello", target.Redirect{Abs: true, Path: "world"}, mss{
		"/":         "",
		"/hello":    "/world",
		"/hello/hi": "",
	}},
	{"/hello", target.Redirect{Pre: true}, mss{
		"/":         "",
		"/hello":    "/",
		"/hello/hi": "/hi",
	}},
	{"/hello", target.Redirect{Pre: true, Path: "world"}, mss{
		"/":         "",
		"/hello":    "/world",
		"/hello/hi": "/world/hi",
	}},
	{"/hello", target.Redirect{Pre: true, Abs: true}, mss{
		"/":         "",
		"/hello":    "/",
		"/hello/hi": "/",
	}},
	{"/hello", target.Redirect{Pre: true, Abs: true, Path: "world"}, mss{
		"/":         "",
		"/hello":    "/world",
		"/hello/hi": "/world",
	}},
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

func TestRouter_AddRedirect(t *testing.T) {
	for _, i := range redirectTests {
		r := New()
		dst := i.dst
		dst.Host = "example.com"
		dst.Code = http.StatusFound
		t.Logf("Running tests for %#v\n", dst)
		r.AddRedirect("www.example.com", i.path, dst)
		for k, v := range i.tests {
			u1 := &url.URL{Scheme: "https", Host: "example.com", Path: v}
			if v == "" {
				u1 = nil
			} else if v == "/" {
				u1.Path = ""
			}
			u2 := &url.URL{Scheme: "https", Host: "www.example.com", Path: k}
			assertHttpRedirect(t, r, http.StatusFound, outputUrl(u1), http.MethodGet, outputUrl(u2))
		}
	}
}

func outputUrl(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.String()
}
