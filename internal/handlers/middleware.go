package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/onedr0p/exportarr/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type wrappedResponseWriter struct {
	inner http.ResponseWriter
	code  int
}

func (w *wrappedResponseWriter) Header() http.Header {
	return w.inner.Header()
}

func (w *wrappedResponseWriter) Write(b []byte) (int, error) {
	return w.inner.Write(b)
}

func (w *wrappedResponseWriter) WriteHeader(code int) {
	w.code = code
	w.inner.WriteHeader(code)
}

func (w *wrappedResponseWriter) Code() int {
	if w.code == 0 {
		return http.StatusOK
	}
	return w.code
}

// LogHandler logs each request at debug level once it completes.
func LogHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := &wrappedResponseWriter{inner: w}
		defer func() {
			slog.Debug("Request Received",
				"remote_addr", r.RemoteAddr,
				"status", ww.Code(),
				"method", r.Method,
				"url", r.URL)
		}()
		handler.ServeHTTP(ww, r)
	})
}

// RecoveryHandler recovers from panics in downstream handlers and returns a 500.
func RecoveryHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// MetricsHandler records scrape duration and request-count metrics around the
// wrapped handler.
func MetricsHandler(conf *config.Config, reg *prometheus.Registry, next http.Handler) http.Handler {
	var (
		scrapeDuration = promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
			Namespace:   conf.App,
			Name:        "scrape_duration_seconds",
			Help:        "Distribution of scrape durations.",
			ConstLabels: prometheus.Labels{"url": conf.URL},
			Buckets:     []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
			// Also expose a sparse native histogram to scrapers that negotiate
			// it; classic buckets above remain for everyone else.
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		})
		requestCount = promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace:   conf.App,
			Name:        "scrape_requests_total",
			Help:        "Total number of HTTP requests made.",
			ConstLabels: prometheus.Labels{"url": conf.URL},
		}, []string{"code"})
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &wrappedResponseWriter{inner: w}
		defer func() {
			scrapeDuration.Observe(time.Since(start).Seconds())
			requestCount.WithLabelValues(strconv.Itoa(ww.Code())).Inc()
		}()
		next.ServeHTTP(ww, r)
	})
}
