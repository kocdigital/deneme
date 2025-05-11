package metrics

import (
    "net/http"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    requestCount = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "route"},
    )
)

func init() {
    prometheus.MustRegister(requestCount)
}

func Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        route := r.URL.Path
        method := r.Method

        requestCount.WithLabelValues(method, route).Inc()
        next.ServeHTTP(w, r)
    })
}

func MetricsHandler() http.Handler {
    return promhttp.Handler()
}