package middleware

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.5, 1.0},
		},
		[]string{"method", "path"},
	)

	businessOpsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "business_operations_total",
			Help: "Total number of business operations",
		},
		[]string{"operation", "result"},
	)
)

// Metrics はPrometheusメトリクスミドルウェア
func Metrics() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(c.Response().Status)
			path := c.Path() // テンプレートパスを使用（カーディナリティ制御）
			if path == "" {
				path = c.Request().URL.Path
			}

			httpRequestsTotal.WithLabelValues(c.Request().Method, path, status).Inc()
			httpRequestDuration.WithLabelValues(c.Request().Method, path).Observe(duration)

			return err
		}
	}
}

// RecordBusinessOp はビジネスオペレーションを記録する
func RecordBusinessOp(operation, result string) {
	businessOpsTotal.WithLabelValues(operation, result).Inc()
}
