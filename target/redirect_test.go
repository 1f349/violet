package target

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirect_FullHost(t *testing.T) {
	assert.Equal(t, "localhost", Redirect{Host: "localhost"}.FullHost())
	assert.Equal(t, "localhost:22", Redirect{Host: "localhost", Port: 22}.FullHost())
}

func TestRedirect_ServeHTTP(t *testing.T) {
	a := []struct {
		Redirect
		target string
	}{
		{Redirect{Host: "example.com", Path: "/bye", Abs: true, Code: http.StatusFound}, "https://example.com/bye"},
		{Redirect{Host: "example.com", Path: "/bye", Code: http.StatusFound}, "https://example.com/bye/hello/world"},
	}
	for _, i := range a {
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "https://www.example.com/hello/world", nil)
		i.ServeHTTP(res, req)
		assert.Equal(t, i.Code, res.Code)
		assert.Equal(t, i.target, res.Header().Get("Location"))
	}
}
