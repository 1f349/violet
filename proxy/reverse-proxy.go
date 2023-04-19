package proxy

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"golang.org/x/net/proxy"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
)

type reverseProxyHostKey int

type ReverseProxyContext interface {
	IsIgnoreCert() bool
	UpdateHeaders(http.Header)
}

func SetReverseProxyHost(req *http.Request, hf ReverseProxyContext) *http.Request {
	ctx := req.Context()
	ctx2 := context.WithValue(ctx, reverseProxyHostKey(0), hf)
	return req.WithContext(ctx2)
}

func CreateHybridReverseProxy() *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director:       func(req *http.Request) {},
		Transport:      NewHybridTransport(),
		ModifyResponse: func(rw *http.Response) error { return nil },
		ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
			log.Printf("[ReverseProxy] Request: %#v\n            -- Error: %s\n", req, err)
			rw.WriteHeader(http.StatusBadGateway)
			_, _ = rw.Write([]byte("502 Bad gateway\n"))
		},
	}
}

type HybridTransport struct {
	baseDialer        *net.Dialer
	normalTransport   http.RoundTripper
	insecureTransport http.RoundTripper
	socksSync         *sync.RWMutex
	socksTransport    map[string]http.RoundTripper
}

func NewHybridTransport() *HybridTransport {
	h := &HybridTransport{
		baseDialer: &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
		socksSync:      &sync.RWMutex{},
		socksTransport: make(map[string]http.RoundTripper),
	}
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
	return h
}

func (h *HybridTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newHost := req.Context().Value(reverseProxyHostKey(0))
	hf, ok := newHost.(ReverseProxyContext)
	if !ok {
		return nil, errors.New("failed to detect reverse proxy configuration")
	}

	// Do a round trip using existing transports
	var trip *http.Response
	var err error
	if hf.IsIgnoreCert() {
		trip, err = h.insecureTransport.RoundTrip(req)
	} else {
		trip, err = h.normalTransport.RoundTrip(req)
	}
	if err != nil {
		return nil, err
	}

	// Override headers
	hf.UpdateHeaders(trip.Header)
	return trip, nil
}

func (h *HybridTransport) getSocksProxy(addr string, insecure bool) (http.RoundTripper, error) {
	if insecure {
		addr = "%i-" + addr
	}
	h.socksSync.RLock()
	s, ok := h.socksTransport[addr]
	h.socksSync.RUnlock()
	if ok {
		return s, nil
	}

	dialer, err := proxy.SOCKS5("tcp", addr, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to the proxy: %s", err)
	}

	if f, ok := dialer.(proxy.ContextDialer); ok {
		t := &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           f.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          15,
			TLSHandshakeTimeout:   10 * time.Second,
			IdleConnTimeout:       30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			DisableKeepAlives:     true,
		}
		if insecure {
			t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
		h.socksSync.Lock()
		h.socksTransport[addr] = t
		h.socksSync.Unlock()
		return t, nil
	}
	return nil, errors.New("cannot create socks5 dialer")
}
