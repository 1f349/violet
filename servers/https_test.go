package servers

import (
	"database/sql"
	"github.com/MrMelon54/violet/certs"
	"github.com/MrMelon54/violet/proxy"
	"github.com/MrMelon54/violet/router"
	"github.com/MrMelon54/violet/servers/conf"
	"github.com/MrMelon54/violet/utils/fake"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type fakeTransport struct{}

func (f *fakeTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	rec.WriteHeader(http.StatusOK)
	return rec.Result(), nil
}

func TestNewHttpsServer_RateLimit(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	assert.NoError(t, err)

	ft := &fakeTransport{}
	httpsConf := &conf.Conf{
		RateLimit: 5,
		Domains:   &fake.Domains{},
		Certs:     certs.New(nil, nil, true),
		Signer:    fake.SnakeOilProv,
		Router:    router.NewManager(db, proxy.NewHybridTransportWithCalls(ft, ft)),
	}
	srv := NewHttpsServer(httpsConf)

	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	req.RemoteAddr = "127.0.0.1:1447"
	assert.NoError(t, err)

	wg := &sync.WaitGroup{}
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			srv.Handler.ServeHTTP(rec, req)
			res := rec.Result()
			assert.Equal(t, http.StatusTeapot, res.StatusCode)
		}()
	}
	wg.Wait()

	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	res := rec.Result()
	assert.Equal(t, http.StatusTooManyRequests, res.StatusCode)
}
