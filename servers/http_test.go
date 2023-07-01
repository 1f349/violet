package servers

import (
	"bytes"
	"github.com/MrMelon54/violet/utils"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewHttpServer_AcmeChallenge(t *testing.T) {
	httpConf := &Conf{
		Domains: &fakeDomains{},
		Acme:    utils.NewAcmeChallenge(),
		Signer:  snakeOilProv,
	}
	srv := NewHttpServer(httpConf)
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
