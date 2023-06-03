package proxy

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"
)

type HybridTransport struct {
	baseDialer        *net.Dialer
	normalTransport   http.RoundTripper
	insecureTransport http.RoundTripper
	socksSync         *sync.RWMutex
	socksTransport    map[string]http.RoundTripper
}

// NewHybridTransport creates a new hybrid transport
func NewHybridTransport() *HybridTransport {
	return NewHybridTransportWithCalls(nil, nil)
}

// NewHybridTransportWithCalls creates new hybrid transport with custom normal
// and insecure http.RoundTripper functions.
//
// NewHybridTransportWithCalls(nil, nil) is equivalent to NewHybridTransport()
func NewHybridTransportWithCalls(normal, insecure http.RoundTripper) *HybridTransport {
	h := &HybridTransport{
		baseDialer: &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
		normalTransport:   normal,
		insecureTransport: insecure,
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
		}
	}
	return h
}

// SecureRoundTrip calls the secure transport
func (h *HybridTransport) SecureRoundTrip(req *http.Request) (*http.Response, error) {
	return h.normalTransport.RoundTrip(req)
}

// InsecureRoundTrip calls the insecure transport
func (h *HybridTransport) InsecureRoundTrip(req *http.Request) (*http.Response, error) {
	return h.insecureTransport.RoundTrip(req)
}
