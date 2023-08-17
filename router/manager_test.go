package router

import (
	"database/sql"
	"github.com/1f349/violet/proxy"
	"github.com/1f349/violet/proxy/websocket"
	"github.com/1f349/violet/target"
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
	ht := proxy.NewHybridTransportWithCalls(ft, ft, &websocket.Server{})
	m := NewManager(db, ht)
	assert.NoError(t, m.internalCompile(m.r))

	rec := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "https://test.example.com", nil)
	assert.NoError(t, err)

	m.ServeHTTP(rec, req)
	res := rec.Result()
	assert.Equal(t, http.StatusTeapot, res.StatusCode)
	assert.Nil(t, ft.req)

	_, err = db.Exec(`INSERT INTO routes (source, destination, flags, active) VALUES (?,?,?,1)`, "*.example.com", "127.0.0.1:8080", target.FlagAbs|target.FlagForwardHost|target.FlagForwardAddr)
	assert.NoError(t, err)

	assert.NoError(t, m.internalCompile(m.r))

	rec = httptest.NewRecorder()
	m.ServeHTTP(rec, req)
	res = rec.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.NotNil(t, ft.req)
}
