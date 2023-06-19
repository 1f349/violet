package servers

import (
	"crypto/rand"
	"crypto/rsa"
	"github.com/MrMelon54/mjwt"
	"github.com/MrMelon54/mjwt/auth"
	"github.com/MrMelon54/mjwt/claims"
	"github.com/MrMelon54/violet/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var snakeOilProv = genSnakeOilProv()

type fakeDomains struct{}

func (f *fakeDomains) IsValid(host string) bool { return host == "example.com" }

func genSnakeOilProv() mjwt.Signer {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	return mjwt.NewMJwtSigner("violet.test", key)
}

func genSnakeOilKey(perm string) string {
	p := claims.NewPermStorage()
	p.Set(perm)
	val, err := snakeOilProv.GenerateJwt("abc", "abc", 5*time.Minute, auth.AccessTokenClaims{
		UserId: 1,
		Perms:  p,
	})
	if err != nil {
		panic(err)
	}
	return val
}

type fakeCompilable struct{ done bool }

func (f *fakeCompilable) Compile() { f.done = true }

var _ utils.Compilable = &fakeCompilable{}

func TestNewApiServer_Compile(t *testing.T) {
	apiConf := &Conf{
		Domains: &fakeDomains{},
		Acme:    utils.NewAcmeChallenge(),
		Verify:  snakeOilProv,
	}
	f := &fakeCompilable{}
	srv := NewApiServer(apiConf, utils.MultiCompilable{f})

	req, err := http.NewRequest(http.MethodPost, "https://example.com/compile", nil)
	assert.NoError(t, err)

	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	res := rec.Result()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
	assert.False(t, f.done)

	req.Header.Set("Authorization", "Bearer "+genSnakeOilKey("violet:compile"))

	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	res = rec.Result()
	assert.Equal(t, http.StatusAccepted, res.StatusCode)
	assert.True(t, f.done)
}

func TestNewApiServer_AcmeChallenge_Put(t *testing.T) {
	apiConf := &Conf{
		Domains: &fakeDomains{},
		Acme:    utils.NewAcmeChallenge(),
		Verify:  snakeOilProv,
	}
	srv := NewApiServer(apiConf, utils.MultiCompilable{})
	acmeKey := genSnakeOilKey("violet:acme-challenge")

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
	apiConf := &Conf{
		Domains: &fakeDomains{},
		Acme:    utils.NewAcmeChallenge(),
		Verify:  snakeOilProv,
	}
	srv := NewApiServer(apiConf, utils.MultiCompilable{})
	acmeKey := genSnakeOilKey("violet:acme-challenge")

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
