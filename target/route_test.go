package target

import (
	"bytes"
	"github.com/1f349/violet/proxy"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type proxyTester struct {
	got bool
	req *http.Request
}

func (p *proxyTester) makeHybridTransport() *proxy.HybridTransport {
	return proxy.NewHybridTransportWithCalls(p, p)
}

func (p *proxyTester) RoundTrip(req *http.Request) (*http.Response, error) {
	p.got = true
	p.req = req
	return &http.Response{StatusCode: http.StatusOK}, nil
}

func TestRoute_HasFlag(t *testing.T) {
	assert.True(t, Route{Flags: FlagPre | FlagAbs}.HasFlag(FlagPre))
	assert.False(t, Route{Flags: FlagPre | FlagAbs}.HasFlag(FlagCors))
}

func TestRoute_ServeHTTP(t *testing.T) {
	a := []struct {
		Route
		target string
	}{
		{Route{Dst: "localhost:1234/bye", Flags: FlagAbs}, "http://localhost:1234/bye"},
		{Route{Dst: "1.2.3.4/bye"}, "http://1.2.3.4/bye/hello/world"},
		{Route{Dst: "2.2.2.2/world", Flags: FlagAbs | FlagSecureMode}, "https://2.2.2.2/world"},
		{Route{Dst: "api.example.com/world", Flags: FlagAbs | FlagSecureMode | FlagForwardHost}, "https://api.example.com/world"},
		{Route{Dst: "api.example.org/world", Flags: FlagAbs | FlagSecureMode | FlagForwardAddr}, "https://api.example.org/world"},
		{Route{Dst: "3.3.3.3/headers", Flags: FlagAbs, Headers: http.Header{"X-Other": []string{"test value"}}}, "http://3.3.3.3/headers"},
	}
	for _, i := range a {
		pt := &proxyTester{}
		i.Proxy = pt.makeHybridTransport()
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "https://www.example.com/hello/world", nil)
		i.ServeHTTP(res, req)

		assert.True(t, pt.got)
		assert.Equal(t, i.target, pt.req.URL.String())
		if i.HasFlag(FlagForwardAddr) {
			assert.Equal(t, req.RemoteAddr, pt.req.Header.Get("X-Forwarded-For"))
		}
		if i.HasFlag(FlagForwardHost) {
			assert.Equal(t, req.Host, pt.req.Host)
		}
		if i.Headers != nil {
			assert.Equal(t, i.Headers, pt.req.Header)
		}
	}
}

func TestRoute_ServeHTTP_Cors(t *testing.T) {
	pt := &proxyTester{}
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "https://www.example.com/test", nil)
	req.Header.Set("Origin", "https://test.example.com")
	i := &Route{Dst: "1.1.1.1:8080/hello", Flags: FlagCors, Proxy: pt.makeHybridTransport()}
	i.ServeHTTP(res, req)

	assert.True(t, pt.got)
	assert.Equal(t, http.MethodOptions, pt.req.Method)
	assert.Equal(t, "http://1.1.1.1:8080/hello/test", pt.req.URL.String())
	assert.Equal(t, "Origin", res.Header().Get("Vary"))
	assert.Equal(t, "*", res.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", res.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "Origin", res.Header().Get("Vary"))
}

func TestRoute_ServeHTTP_Body(t *testing.T) {
	pt := &proxyTester{}
	res := httptest.NewRecorder()
	buf := bytes.NewBuffer([]byte{0x54})
	req := httptest.NewRequest(http.MethodPost, "https://www.example.com/test", buf)
	req.Header.Set("Origin", "https://test.example.com")
	i := &Route{Dst: "1.1.1.1:8080/hello", Flags: FlagCors, Proxy: pt.makeHybridTransport()}
	i.ServeHTTP(res, req)

	assert.True(t, pt.got)
	assert.Equal(t, http.MethodPost, pt.req.Method)
	assert.Equal(t, "http://1.1.1.1:8080/hello/test", pt.req.URL.String())
	all, err := io.ReadAll(pt.req.Body)
	assert.NoError(t, err)
	assert.Equal(t, 0, bytes.Compare(all, []byte{0x54}))
	assert.NoError(t, pt.req.Body.Close())
}
