package error_pages

import (
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrorPages_ServeError(t *testing.T) {
	errorPages := New(nil)

	rec := httptest.NewRecorder()
	errorPages.ServeError(rec, http.StatusTeapot)
	res := rec.Result()
	assert.Equal(t, http.StatusTeapot, res.StatusCode)
	assert.Equal(t, "418 I'm a teapot", res.Status)
	a, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.Equal(t, "418 I'm a teapot\n\n", string(a))

	rec = httptest.NewRecorder()
	errorPages.ServeError(rec, 469)
	res = rec.Result()
	assert.Equal(t, 469, res.StatusCode)
	assert.Equal(t, "469 ", res.Status)
	a, err = io.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.Equal(t, "469 Unknown Error Code\n\n", string(a))
}
