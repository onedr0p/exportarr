package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/onedr0p/exportarr/internal/assert"
	"github.com/onedr0p/exportarr/internal/config"
)

func TestHealthzHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	HealthzHandler(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Equal(t, rec.Body.String(), "OK")
}

func TestIndexHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	IndexHandler(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Contains(t, rec.Body.String(), "/metrics")
}

func TestRecoveryHandler(t *testing.T) {
	h := RecoveryHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	assert.NotPanics(t, func() {
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	})
	assert.Equal(t, rec.Code, http.StatusInternalServerError)
}

func TestLogHandler_PassesThroughStatusAndBody(t *testing.T) {
	h := LogHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("nope"))
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/missing", nil))

	assert.Equal(t, rec.Code, http.StatusNotFound)
	assert.Equal(t, rec.Body.String(), "nope")
}

// labelValue returns the value of the named label on the first metric of the
// named family, failing the test if the family is not in mfs.
func labelValue(t *testing.T, reg *prometheus.Registry, family, label string) string {
	t.Helper()
	mfs, err := reg.Gather()
	assert.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != family {
			continue
		}
		for _, lp := range mf.GetMetric()[0].GetLabel() {
			if lp.GetName() == label {
				return lp.GetValue()
			}
		}
	}
	t.Fatalf("metric family %q with label %q not gathered", family, label)
	return ""
}

func TestMetricsHandler_RecordsDurationAndStatusCode(t *testing.T) {
	conf := &config.Config{App: "radarr", URL: "http://radarr:7878"}
	reg := prometheus.NewRegistry()
	h := MetricsHandler(conf, reg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	// The wrapped writer must pass the inner handler's status through.
	assert.Equal(t, rec.Code, http.StatusTeapot)
	// The counter labels the request with the inner handler's status code.
	assert.Equal(t, labelValue(t, reg, "radarr_scrape_requests_total", "code"), "418")

	mfs, err := reg.Gather()
	assert.NoError(t, err)
	var sawDuration bool
	for _, mf := range mfs {
		if mf.GetName() == "radarr_scrape_duration_seconds" {
			sawDuration = true
			assert.Equal(t, mf.GetMetric()[0].GetHistogram().GetSampleCount(), uint64(1))
		}
	}
	assert.True(t, sawDuration, "radarr_scrape_duration_seconds not gathered")
}

func TestMetricsHandler_DefaultsToStatusOK(t *testing.T) {
	conf := &config.Config{App: "sonarr", URL: "http://sonarr:8989"}
	reg := prometheus.NewRegistry()
	h := MetricsHandler(conf, reg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok")) // no explicit WriteHeader
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	assert.Equal(t, labelValue(t, reg, "sonarr_scrape_requests_total", "code"), "200")
}
