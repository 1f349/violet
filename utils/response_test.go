package utils

import (
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRespondHttpStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	RespondHttpStatus(rec, http.StatusTeapot)
	res := rec.Result()
	assert.Equal(t, http.StatusTeapot, res.StatusCode)
	assert.Equal(t, "418 I'm a teapot", res.Status)
	a, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.Equal(t, "418 I'm a teapot\n", string(a))
}

func TestRespondVioletError(t *testing.T) {
	rec := httptest.NewRecorder()
	RespondVioletError(rec, http.StatusTeapot, "Hidden Error Message")
	res := rec.Result()
	assert.Equal(t, http.StatusTeapot, res.StatusCode)
	assert.Equal(t, "418 I'm a teapot", res.Status)
	a, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.Equal(t, "418 I'm a teapot\n", string(a))
	assert.Equal(t, "Hidden Error Message", res.Header.Get("X-Violet-Error"))
}
