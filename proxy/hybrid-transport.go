package proxy

import (
	"crypto/tls"
	"github.com/1f349/violet/logger"
	"github.com/1f349/violet/proxy/websocket"
	"github.com/google/uuid"
	"net"
	"net/http"
	"sync"
	"time"
)

var loggerSecure = logger.Logger.WithPrefix("Violet Secure Transport")
var loggerInsecure = logger.Logger.WithPrefix("Violet Insecure Transport")
var loggerWebsocket = logger.Logger.WithPrefix("Violet Websocket Transport")

type HybridTransport struct {
	baseDialer        *net.Dialer
	normalTransport   http.RoundTripper
	insecureTransport http.RoundTripper
	socksSync         *sync.RWMutex
	socksTransport    map[string]http.RoundTripper
	ws                *websocket.Server
}

// NewHybridTransport creates a new hybrid transport
func NewHybridTransport(ws *websocket.Server) *HybridTransport {
	return NewHybridTransportWithCalls(nil, nil, ws)
}

// NewHybridTransportWithCalls creates new hybrid transport with custom normal
// and insecure http.RoundTripper functions.
//
// NewHybridTransportWithCalls(nil, nil) is equivalent to NewHybridTransport()
func NewHybridTransportWithCalls(normal, insecure http.RoundTripper, ws *websocket.Server) *HybridTransport {
	h := &HybridTransport{
		baseDialer: &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
		normalTransport:   normal,
		insecureTransport: insecure,
		ws:                ws,
	}
	if h.normalTransport == nil {
		h.normalTransport = &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           h.baseDialer.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          15,
			TLSHandshakeTimeout:   10 * time.Second,
			IdleConnTimeout:       30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			DisableKeepAlives:     true,
		}
	}
	if h.insecureTransport == nil {
		h.insecureTransport = &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           h.baseDialer.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          15,
			TLSHandshakeTimeout:   10 * time.Second,
			IdleConnTimeout:       30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives:     true,
		}
	}
	return h
}

// SecureRoundTrip calls the secure transport
func (h *HybridTransport) SecureRoundTrip(req *http.Request) (*http.Response, error) {
	u := uuid.New()
	loggerSecure.Info("Start upgrade", "id", u)
	defer loggerSecure.Info("Stop upgrade", "id", u)
	return h.normalTransport.RoundTrip(req)
}

// InsecureRoundTrip calls the insecure transport
func (h *HybridTransport) InsecureRoundTrip(req *http.Request) (*http.Response, error) {
	u := uuid.New()
	loggerInsecure.Info("Start upgrade", "id", u)
	defer loggerInsecure.Info("Stop upgrade", "id", u)
	return h.insecureTransport.RoundTrip(req)
}

// ConnectWebsocket calls the websocket upgrader and thus hijacks the connection
func (h *HybridTransport) ConnectWebsocket(rw http.ResponseWriter, req *http.Request) {
	u := uuid.New()
	loggerWebsocket.Info("Start upgrade", "id", u)
	h.ws.Upgrade(rw, req)
	loggerWebsocket.Info("Stop upgrade", "id", u)
}
