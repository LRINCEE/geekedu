package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests handled by the service.",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
	grpcClientRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_client_requests_total",
			Help: "Total gRPC client requests.",
		},
		[]string{"service", "method", "code"},
	)
	grpcClientRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_client_request_duration_seconds",
			Help:    "gRPC client request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "code"},
	)
	grpcServerRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_server_requests_total",
			Help: "Total gRPC server requests.",
		},
		[]string{"service", "method", "code"},
	)
	grpcServerRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_server_request_duration_seconds",
			Help:    "gRPC server request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "code"},
	)
	cacheRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_requests_total",
			Help: "Total cache operations.",
		},
		[]string{"component", "op", "result"},
	)
	redisCircuitOpen = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "redis_circuit_open",
			Help: "Whether the Redis circuit breaker is currently open.",
		},
		[]string{"component"},
	)
	redisCircuitOpenTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_circuit_open_total",
			Help: "Total number of times the Redis circuit breaker opened.",
		},
		[]string{"component"},
	)
	courseListLoadTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "course_list_load_total",
			Help: "Total number of course list loads by source.",
		},
		[]string{"source"},
	)
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDurationSeconds,
		grpcClientRequestsTotal,
		grpcClientRequestDurationSeconds,
		grpcServerRequestsTotal,
		grpcServerRequestDurationSeconds,
		cacheRequestsTotal,
		redisCircuitOpen,
		redisCircuitOpenTotal,
		courseListLoadTotal,
	)
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

func ObserveHTTPRequest(method, path string, status int, duration time.Duration) {
	code := strconv.Itoa(status)
	httpRequestsTotal.WithLabelValues(method, path, code).Inc()
	httpRequestDurationSeconds.WithLabelValues(method, path, code).Observe(duration.Seconds())
}

func ObserveGRPCClient(service, method, code string, duration time.Duration) {
	grpcClientRequestsTotal.WithLabelValues(service, method, code).Inc()
	grpcClientRequestDurationSeconds.WithLabelValues(service, method, code).Observe(duration.Seconds())
}

func ObserveGRPCServer(service, method, code string, duration time.Duration) {
	grpcServerRequestsTotal.WithLabelValues(service, method, code).Inc()
	grpcServerRequestDurationSeconds.WithLabelValues(service, method, code).Observe(duration.Seconds())
}

func ObserveCacheRequest(component, op, result string) {
	cacheRequestsTotal.WithLabelValues(component, op, result).Inc()
}

func SetRedisCircuitOpen(component string, open bool) {
	if open {
		redisCircuitOpen.WithLabelValues(component).Set(1)
		return
	}
	redisCircuitOpen.WithLabelValues(component).Set(0)
}

func IncRedisCircuitOpen(component string) {
	redisCircuitOpenTotal.WithLabelValues(component).Inc()
}

func IncCourseListLoad(source string) {
	courseListLoadTotal.WithLabelValues(source).Inc()
}
