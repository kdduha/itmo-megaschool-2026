package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	namespace = "backend"

	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "code"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	filePreprocessTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "file_preprocess_total",
			Help:      "Number of preprocess files",
		},
		[]string{"status", "file_format"},
	)

	filePreprocessDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "file_preprocess_duration",
			Help:      "Number of preprocess files",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"status", "file_format"},
	)
)

func HttpRequestsTotal(method, path, code string) {
	httpRequestsTotal.With(prometheus.Labels{
		"method": method,
		"path":   path,
		"code":   code,
	}).Inc()
}

func HttpRequestDuration(method, path string, duration time.Duration) {
	httpRequestDuration.With(prometheus.Labels{
		"method": method,
		"path":   path,
	}).Observe(duration.Seconds())
}

func FilePreprocessTotal(status, fileFormat string) {
	filePreprocessTotal.With(prometheus.Labels{
		"status":      status,
		"file_format": fileFormat,
	}).Inc()
}

func FilePreprocessDuration(status, fileFormat string, duration time.Duration) {
	filePreprocessDuration.With(prometheus.Labels{
		"status":      status,
		"file_format": fileFormat,
	}).Observe(duration.Seconds())
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := &statusResponseWriter{w, 200}
		next.ServeHTTP(w, r)

		duration := time.Since(start)
		HttpRequestsTotal(r.Method, r.URL.Path, http.StatusText(ww.status))
		HttpRequestDuration(r.Method, r.URL.Path, duration)
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
