package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeResponseWriter struct{ h http.Header }

func (f fakeResponseWriter) Header() http.Header             { return f.h }
func (f fakeResponseWriter) Write(bytes []byte) (int, error) { return len(bytes), nil }
func (f fakeResponseWriter) WriteHeader(statusCode int)      {}

func BenchmarkRedirect(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	res := &fakeResponseWriter{h: make(http.Header, 10)}
	req := httptest.NewRequest(http.MethodGet, "https://www.example.com", nil)
	for i := 0; i < b.N; i++ {
		http.Redirect(res, req, "https://example.com", http.StatusPermanentRedirect)
	}
}

func BenchmarkFastRedirect(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	res := &fakeResponseWriter{h: make(http.Header, 10)}
	req := httptest.NewRequest(http.MethodGet, "https://www.example.com", nil)
	for i := 0; i < b.N; i++ {
		FastRedirect(res, req, "https://example.com", http.StatusPermanentRedirect)
	}
}
