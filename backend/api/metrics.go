package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// RequestCounter tracks total number of HTTP requests
	RequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// RequestDuration tracks duration of HTTP requests
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// ActiveConnections tracks number of active WebSocket connections
	ActiveConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "websocket_active_connections",
			Help: "Number of active WebSocket connections",
		},
	)
)

func init() {
	// Register all metrics with Prometheus
	prometheus.MustRegister(RequestCounter)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(ActiveConnections)
}

// MetricsHandler returns the Prometheus metrics endpoint handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
