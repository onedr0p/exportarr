package handlers

import (
	"fmt"
	"github.com/shamelin/exportarr/internal/config"
	"net/http"
	"time"

	"go.uber.org/zap"

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

// Log internal request to stdout
func LogHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := &wrappedResponseWriter{inner: w}
		defer func() {
			zap.S().Debugw("Request Received",
				"remote_addr", r.RemoteAddr,
				"status", ww.Code(),
				"method", r.Method,
				"url", r.URL)
		}()
		handler.ServeHTTP(w, r)
	})
}

func RecoveryHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				zap.S().Errorw("panic recovered", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func MetricsHandler(conf *config.Config, reg *prometheus.Registry, next http.Handler) http.Handler {
	var (
		scrapDuration = promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Namespace:   conf.App,
			Name:        "scrape_duration_seconds",
			Help:        "Duration of the last scrape of metrics from Exportarr.",
			ConstLabels: prometheus.Labels{"url": conf.URL},
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
			scrapDuration.Set(time.Since(start).Seconds())
			requestCount.WithLabelValues(fmt.Sprintf("%d", ww.Code())).Inc()
		}()
		next.ServeHTTP(ww, r)
	})
}
