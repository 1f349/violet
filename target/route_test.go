package target

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

type proxyTester struct {
	got bool
	rw  http.ResponseWriter
	req *http.Request
}

func (p *proxyTester) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.got = true
	p.rw = rw
	p.req = req
}

func TestRoute_FullHost(t *testing.T) {
	assert.Equal(t, "localhost", Route{Host: "localhost"}.FullHost())
	assert.Equal(t, "localhost:22", Route{Host: "localhost", Port: 22}.FullHost())
}

func TestRoute_ServeHTTP(t *testing.T) {
	a := []struct {
		Route
		target string
	}{
		{Route{Host: "localhost", Port: 1234, Path: "/bye", Abs: true}, "http://localhost:1234/bye"},
		{Route{Host: "1.2.3.4", Path: "/bye"}, "http://1.2.3.4:80/bye/hello/world"},
		{Route{Host: "2.2.2.2", Path: "/world", Abs: true, SecureMode: true}, "https://2.2.2.2:443/world"},
		{Route{Host: "api.example.com", Path: "/world", Abs: true, SecureMode: true, ForwardHost: true}, "https://api.example.com:443/world"},
		{Route{Host: "api.example.org", Path: "/world", Abs: true, SecureMode: true, ForwardAddr: true}, "https://api.example.org:443/world"},
		{Route{Host: "3.3.3.3", Path: "/headers", Abs: true, Headers: http.Header{"X-Other": []string{"test value"}}}, "http://3.3.3.3:80/headers"},
	}
	for _, i := range a {
		pt := &proxyTester{}
		i.Proxy = pt
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "https://www.example.com/hello/world", nil)
		i.ServeHTTP(res, req)

		assert.True(t, pt.got)
		assert.Equal(t, i.target, pt.req.URL.String())
		if i.ForwardAddr {
			assert.Equal(t, req.RemoteAddr, pt.req.Header.Get("X-Forwarded-For"))
		}
		if i.ForwardHost {
			assert.Equal(t, req.Host, pt.req.Host)
		}
		if i.Headers != nil {
			assert.Equal(t, i.Headers, pt.req.Header)
		}
	}
}

func TestRoute_ServeHTTP_Cors(t *testing.T) {
	pt := &proxyTester{}
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "https://www.example.com/test", nil)
	req.Header.Set("Origin", "https://test.example.com")
	i := &Route{Host: "1.1.1.1", Port: 8080, Path: "/hello", Cors: true, Proxy: pt}
	i.ServeHTTP(res, req)

	assert.True(t, pt.got)
	assert.Equal(t, http.MethodOptions, pt.req.Method)
	assert.Equal(t, "http://1.1.1.1:8080/hello/test", pt.req.URL.String())
	assert.Equal(t, "Origin", res.Header().Get("Vary"))
	assert.Equal(t, "*", res.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", res.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "Origin", res.Header().Get("Vary"))
}
