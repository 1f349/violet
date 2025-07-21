package api

import (
	"github.com/1f349/violet/servers/conf"
	"github.com/1f349/violet/utils"
	"github.com/1f349/violet/utils/fake"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewApiServer_Compile(t *testing.T) {
	apiConf := &conf.Conf{
		Domains: &fake.Domains{},
		Acme:    utils.NewAcmeChallenge(),
		Signer:  fake.SnakeOilProv.KeyStore(),
	}
	f := &fake.Compilable{}
	srv := NewApiServer(apiConf, utils.MultiCompilable{f}, "abc123")

	req, err := http.NewRequest(http.MethodPost, "https://example.com/compile", nil)
	assert.NoError(t, err)

	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	res := rec.Result()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
	assert.False(t, f.Done)

	req.Header.Set("Authorization", "Bearer "+fake.GenSnakeOilKey("violet:compile"))

	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	res = rec.Result()
	assert.Equal(t, http.StatusAccepted, res.StatusCode)
	assert.True(t, f.Done)
}

func TestNewApiServer_AcmeChallenge_Put(t *testing.T) {
	apiConf := &conf.Conf{
		Domains: &fake.Domains{},
		Acme:    utils.NewAcmeChallenge(),
		Signer:  fake.SnakeOilProv.KeyStore(),
	}
	srv := NewApiServer(apiConf, utils.MultiCompilable{}, "abc123")
	acmeKey := fake.GenSnakeOilKey("violet:acme-challenge")

	// Valid domain
	req, err := http.NewRequest(http.MethodPut, "https://example.com/acme-challenge/example.com/123/123abc", nil)
	assert.NoError(t, err)

	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	res := rec.Result()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)

	req.Header.Set("Authorization", "Bearer "+acmeKey)

	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	res = rec.Result()
	assert.Equal(t, http.StatusAccepted, res.StatusCode)
	assert.Equal(t, "123abc", apiConf.Acme.Get("example.com", "123"))

	// Invalid domain
	req, err = http.NewRequest(http.MethodPut, "https://example.com/acme-challenge/notexample.com/123/123abc", nil)
	assert.NoError(t, err)

	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	res = rec.Result()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)

	req.Header.Set("Authorization", "Bearer "+acmeKey)

	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	res = rec.Result()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	assert.Equal(t, "Invalid ACME challenge domain", res.Header.Get("X-Violet-Error"))
}

func TestNewApiServer_AcmeChallenge_Delete(t *testing.T) {
	apiConf := &conf.Conf{
		Domains: &fake.Domains{},
		Acme:    utils.NewAcmeChallenge(),
		Signer:  fake.SnakeOilProv.KeyStore(),
	}
	srv := NewApiServer(apiConf, utils.MultiCompilable{}, "abc123")
	acmeKey := fake.GenSnakeOilKey("violet:acme-challenge")

	// Valid domain
	req, err := http.NewRequest(http.MethodDelete, "https://example.com/acme-challenge/example.com/123", nil)
	assert.NoError(t, err)

	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	res := rec.Result()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)

	req.Header.Set("Authorization", "Bearer "+acmeKey)
	apiConf.Acme.Put("example.com", "123", "123abc")

	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	res = rec.Result()
	assert.Equal(t, http.StatusAccepted, res.StatusCode)
	assert.Equal(t, "", apiConf.Acme.Get("example.com", "123"))

	// Invalid domain
	req, err = http.NewRequest(http.MethodDelete, "https://example.com/acme-challenge/notexample.com/123", nil)
	assert.NoError(t, err)

	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	res = rec.Result()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)

	req.Header.Set("Authorization", "Bearer "+acmeKey)

	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	res = rec.Result()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	assert.Equal(t, "Invalid ACME challenge domain", res.Header.Get("X-Violet-Error"))
}
