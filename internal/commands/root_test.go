package commands

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/onedr0p/exportarr/internal/assert"
)

func fakeTarget(path, app, url string, metricName string) metricsTarget {
	return metricsTarget{
		path: path,
		app:  app,
		url:  url,
		register: func(r prometheus.Registerer) {
			r.MustRegister(prometheus.NewGaugeFunc(
				prometheus.GaugeOpts{Name: metricName, Help: "test"},
				func() float64 { return 1 },
			))
		},
	}
}

func TestBuildMux_SingleTarget_MountsBareMetricsPath(t *testing.T) {
	mux := buildMux([]metricsTarget{fakeTarget("", "radarr", "http://radarr", "radarr_probe")})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Contains(t, rec.Body.String(), "radarr_probe")

	// index links to the bare /metrics path
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Contains(t, rec.Body.String(), "href='/metrics'")

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	assert.Equal(t, rec.Code, http.StatusOK)
}

func TestBuildMux_MultiTarget_MountsPerAppPaths(t *testing.T) {
	mux := buildMux([]metricsTarget{
		fakeTarget("radarr", "radarr", "http://radarr", "radarr_probe"),
		fakeTarget("sonarr", "sonarr", "http://sonarr", "sonarr_probe"),
	})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics/radarr", nil))
	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Contains(t, rec.Body.String(), "radarr_probe")
	assert.NotContains(t, rec.Body.String(), "sonarr_probe")

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics/sonarr", nil))
	assert.Equal(t, rec.Code, http.StatusOK)
	assert.Contains(t, rec.Body.String(), "sonarr_probe")
	assert.NotContains(t, rec.Body.String(), "radarr_probe")

	// the bare /metrics path is not mounted in multi-app mode; ServeMux falls
	// through to the "/" index page instead of a dedicated 404
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Equal(t, rec.Code, http.StatusOK)
	assert.NotContains(t, rec.Body.String(), "radarr_probe")
	assert.NotContains(t, rec.Body.String(), "sonarr_probe")

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Contains(t, rec.Body.String(), "href='/metrics/radarr'")
	assert.Contains(t, rec.Body.String(), "href='/metrics/sonarr'")
}

func TestServeHTTP_NoTargets_ReturnsErrorWithoutBinding(t *testing.T) {
	err := serveHTTP(nil)
	assert.Error(t, err)
}

func TestSingleTarget(t *testing.T) {
	cs := []prometheus.Collector{prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{Name: "probe_metric", Help: "test"},
		func() float64 { return 1 },
	)}
	target := singleTarget("radarr", "http://radarr:7878", cs)

	assert.Equal(t, target.path, "")
	assert.Equal(t, target.app, "radarr")
	assert.Equal(t, target.url, "http://radarr:7878")

	reg := prometheus.NewRegistry()
	assert.NotPanics(t, func() { target.register(reg) })
}
