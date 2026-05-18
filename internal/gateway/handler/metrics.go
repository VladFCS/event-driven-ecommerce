package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	gatewayHTTPRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_http_requests_total",
			Help: "Total number of HTTP requests handled by gateway-service.",
		},
		[]string{"method", "route", "status_code"},
	)
	gatewayHTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_http_request_duration_seconds",
			Help:    "Duration of HTTP requests handled by gateway-service.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status_code"},
	)
)

func requestMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()

		c.Next()

		labels := []string{
			c.Request.Method,
			metricsRouteLabel(c),
			strconv.Itoa(c.Writer.Status()),
		}

		gatewayHTTPRequestTotal.WithLabelValues(labels...).Inc()
		gatewayHTTPRequestDuration.WithLabelValues(labels...).Observe(time.Since(startedAt).Seconds())
	}
}

func metricsHandler() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}

func metricsRouteLabel(c *gin.Context) string {
	route := c.FullPath()
	if route == "" {
		return "unknown"
	}

	return route
}
