package router

import (
	"database/sql"
	"github.com/MrMelon54/violet/proxy"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeTransport struct{ req *http.Request }

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.req = req
	rec := httptest.NewRecorder()
	rec.WriteHeader(http.StatusOK)
	return rec.Result(), nil
}

func TestNewManager(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	assert.NoError(t, err)

	ft := &fakeTransport{}
	ht := proxy.NewHybridTransportWithCalls(ft, ft)
	m := NewManager(db, ht)
	assert.NoError(t, m.internalCompile(m.r))

	rec := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "https://test.example.com", nil)
	assert.NoError(t, err)

	m.ServeHTTP(rec, req)
	res := rec.Result()
	assert.Equal(t, http.StatusTeapot, res.StatusCode)
	assert.Nil(t, ft.req)

	_, err = db.Exec(`INSERT INTO routes (source, pre, destination, abs, cors, secure_mode, forward_host, forward_addr, ignore_cert, active) VALUES (?,?,?,?,?,?,?,?,?,?)`, "*.example.com", 0, "127.0.0.1:8080", 1, 0, 0, 1, 1, 0, 1)
	assert.NoError(t, err)

	assert.NoError(t, m.internalCompile(m.r))

	rec = httptest.NewRecorder()
	m.ServeHTTP(rec, req)
	res = rec.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.NotNil(t, ft.req)
}
