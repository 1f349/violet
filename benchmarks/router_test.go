package benchmarks

import (
	"github.com/MrMelon54/violet/router"
	"github.com/MrMelon54/violet/target"
	gorillaRouter "github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"testing"
)

func benchRequest(b *testing.B, router http.Handler, r *http.Request) {
	w := httptest.NewRecorder()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, r)
	}
	if w.Header().Get("Location") != "https://example.com" {
		b.Fatal("Location: ", w.Header().Get("Location"), " != https://example.com")
	}
}

func BenchmarkVioletRouter(b *testing.B) {
	r := router.New(nil)
	r.AddRedirect("*.example.com", "", target.Redirect{
		Pre:  true,
		Host: "example.com",
		Code: http.StatusPermanentRedirect,
	})
	benchRequest(b, r, httptest.NewRequest(http.MethodGet, "https://www.example.com", nil))
}

func BenchmarkGorillaMux(b *testing.B) {
	r := gorillaRouter.NewRouter()
	r.Host("{subdomain}.example.com").Handler(target.Redirect{
		Pre:  true,
		Host: "example.com",
		Code: http.StatusPermanentRedirect,
	})
	benchRequest(b, r, httptest.NewRequest(http.MethodGet, "https://www.example.com/", nil))
}
