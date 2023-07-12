package target

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirect_ServeHTTP(t *testing.T) {
	a := []struct {
		Redirect
		target string
	}{
		{Redirect{Dst: "example.com/bye", Flags: FlagAbs, Code: http.StatusFound}, "https://example.com/bye"},
		{Redirect{Dst: "example.com/bye", Code: http.StatusFound}, "https://example.com/bye/hello/world"},
	}
	for _, i := range a {
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "https://www.example.com/hello/world", nil)
		i.ServeHTTP(res, req)
		assert.Equal(t, i.Code, res.Code)
		assert.Equal(t, i.target, res.Header().Get("Location"))
	}
}
