package metrics

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"net/netip"
)

// Copyright 2022 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package metrics is adapted from
// https://github.com/bwplotka/correlator/tree/main/examples/observability/ping/pkg/httpinstrumentation
// https://github.com/prometheus/client_golang/blob/main/examples/middleware/httpmiddleware/httpmiddleware.go

type Middleware interface {
	// WrapHandler wraps the given HTTP handler for instrumentation.
	WrapHandler(handlerName string, handler http.Handler) http.HandlerFunc
}

type middleware struct {
	buckets  []float64
	registry prometheus.Registerer
}

// WrapHandler wraps the given HTTP handler for instrumentation:
// It registers four metric collectors (if not already done) and reports HTTP
// metrics to the (newly or already) registered collectors.
// Each has a constant label named "handler" with the provided handlerName as
// value.
func (m *middleware) WrapHandler(handlerName string, handler http.Handler) http.HandlerFunc {
	reg := prometheus.WrapRegistererWith(prometheus.Labels{"handler": handlerName}, m.registry)

	requestsTotal := promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Tracks the number of HTTP requests.",
		}, []string{"method", "code", "host", "path", "ip"},
	)
	requestDuration := promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Tracks the latencies for HTTP requests.",
			Buckets: m.buckets,
		},
		[]string{"method", "code", "host", "path", "ip"},
	)
	requestSize := promauto.With(reg).NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_request_size_bytes",
			Help: "Tracks the size of HTTP requests.",
		},
		[]string{"method", "code", "host", "path", "ip"},
	)
	responseSize := promauto.With(reg).NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_response_size_bytes",
			Help: "Tracks the size of HTTP responses.",
		},
		[]string{"method", "code", "host", "path", "ip"},
	)
	activeRequests := promauto.With(reg).NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_requests",
			Help: "Number of active connections to the service",
		},
	)

	hostCtxGetter := promhttp.WithLabelFromCtx("host", func(ctx context.Context) string {
		s, _ := ctx.Value(hostCtxKey).(string)
		return s
	})
	ipCtxGetter := promhttp.WithLabelFromCtx("ip", func(ctx context.Context) string {
		s, _ := ctx.Value(ipCtxKey).(netip.AddrPort)
		return s.Addr().String()
	})
	pathCtxGetter := promhttp.WithLabelFromCtx("path", func(ctx context.Context) string {
		s, _ := ctx.Value(pathCtxKey).(string)
		return s
	})

	// Wraps the provided http.Handler to observe the request result with the provided metrics.
	base := promhttp.InstrumentHandlerCounter(
		requestsTotal,
		promhttp.InstrumentHandlerDuration(
			requestDuration,
			promhttp.InstrumentHandlerRequestSize(
				requestSize,
				promhttp.InstrumentHandlerResponseSize(
					responseSize,
					http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
						activeRequests.Inc()
						handler.ServeHTTP(rw, req)
						activeRequests.Dec()
					}),
					hostCtxGetter,
					ipCtxGetter,
					pathCtxGetter,
				),
				hostCtxGetter,
				ipCtxGetter,
				pathCtxGetter,
			),
			hostCtxGetter,
			ipCtxGetter,
			pathCtxGetter,
		),
		hostCtxGetter,
		ipCtxGetter,
		pathCtxGetter,
	)

	return base.ServeHTTP
}

// New returns a Middleware interface.
func New(registry prometheus.Registerer, buckets []float64) Middleware {
	if buckets == nil {
		buckets = prometheus.ExponentialBuckets(0.1, 1.5, 5)
	}

	return &middleware{
		buckets:  buckets,
		registry: registry,
	}
}

type ctxKey uint8

const (
	hostCtxKey ctxKey = iota
	ipCtxKey
	pathCtxKey
)

func AddMetricsCtx(req *http.Request) *http.Request {
	ctx := req.Context()
	ctx = context.WithValue(ctx, hostCtxKey, req.Host)
	addrPort, err := netip.ParseAddrPort(req.RemoteAddr)
	if err == nil {
		ctx = context.WithValue(ctx, ipCtxKey, addrPort)
	}
	ctx = context.WithValue(ctx, pathCtxKey, req.URL.Path)
	return req.WithContext(ctx)
}
