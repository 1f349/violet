package target

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirect_OnDomain(t *testing.T) {
	assert.True(t, Route{Src: "example.com"}.OnDomain("example.com"))
	assert.True(t, Route{Src: "test.example.com"}.OnDomain("example.com"))
	assert.True(t, Route{Src: "example.com/hello"}.OnDomain("example.com"))
	assert.True(t, Route{Src: "test.example.com/hello"}.OnDomain("example.com"))
	assert.False(t, Route{Src: "example.com"}.OnDomain("example.org"))
	assert.False(t, Route{Src: "test.example.com"}.OnDomain("example.org"))
	assert.False(t, Route{Src: "example.com/hello"}.OnDomain("example.org"))
	assert.False(t, Route{Src: "test.example.com/hello"}.OnDomain("example.org"))
}

func TestRedirect_HasFlag(t *testing.T) {
	assert.True(t, Route{Flags: FlagPre | FlagAbs}.HasFlag(FlagPre))
	assert.False(t, Route{Flags: FlagPre | FlagAbs}.HasFlag(FlagCors))
}

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
		assert.Equal(t, i.Code, int32(res.Code))
		assert.Equal(t, i.target, res.Header().Get("Location"))
	}
}
