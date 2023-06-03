package proxy

import (
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"
)

type customTransport struct{}

func (c *customTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	res := httptest.NewRecorder()
	res.WriteHeader(http.StatusOK)
	res.Write([]byte{0x54, 0x54})
	return res.Result(), nil
}

func SetupReverseProxy() *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director:       func(req *http.Request) {},
		Transport:      &customTransport{},
		ModifyResponse: func(rw *http.Response) error { return nil },
		ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
			log.Printf("[ReverseProxy] Request: %#v\n            -- Error: %s\n", req, err)
			rw.WriteHeader(http.StatusBadGateway)
			_, _ = rw.Write([]byte("502 Bad gateway\n"))
		},
	}
}

func BenchmarkHttpUtilReverseProxy(b *testing.B) {
	rev := SetupReverseProxy()
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		rev.ServeHTTP(rec, req)
	}
}

func BenchmarkCustomTransport(b *testing.B) {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	t := &customTransport{}
	for i := 0; i < b.N; i++ {
		_, _ = t.RoundTrip(req)
	}
}
