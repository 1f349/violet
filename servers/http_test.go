package servers

import (
	"bytes"
	"github.com/1f349/violet/servers/conf"
	"github.com/1f349/violet/utils"
	"github.com/1f349/violet/utils/fake"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewHttpServer_AcmeChallenge(t *testing.T) {
	httpConf := &conf.Conf{
		Domains: &fake.Domains{},
		Acme:    utils.NewAcmeChallenge(),
		Signer:  fake.SnakeOilProv,
	}
	srv := NewHttpServer(443, httpConf, nil)
	httpConf.Acme.Put("example.com", "456", "456def")

	req, err := http.NewRequest(http.MethodGet, "https://example.com/.well-known/acme-challenge/456", nil)
	assert.NoError(t, err)

	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	res := rec.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	all, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.Equal(t, 0, bytes.Compare([]byte("456def"), all))

	// Invalid key
	req, err = http.NewRequest(http.MethodGet, "https://example.com/.well-known/acme-challenge/789", nil)
	assert.NoError(t, err)

	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	res = rec.Result()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)

	all, err = io.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.Equal(t, 0, bytes.Compare([]byte(""), all))
}
