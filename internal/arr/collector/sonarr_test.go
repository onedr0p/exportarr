package collector

import (
	"github.com/onedr0p/exportarr/internal/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	client "github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/fixtures"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

const sonarrTestFixturesPath = "../testdata/sonarr/"

func newTestSonarrServer(t *testing.T, fn func(http.ResponseWriter, *http.Request)) (*httptest.Server, error) {
	return fixtures.NewTestServer(t, sonarrTestFixturesPath, fn)
}

func TestSonarrCollect(t *testing.T) {
	tests := []struct {
		name                string
		config              *config.ArrConfig
		expectedMetricsFile string
	}{
		{
			name: "basic",
			config: &config.ArrConfig{
				App:                   "sonarr",
				APIVersion:            "v3",
				DisableQualityMetrics: true,
				DisableEpisodeMetrics: true,
			},
			expectedMetricsFile: "expected_metrics.txt",
		},
		{
			name: "default_collects_everything",
			config: &config.ArrConfig{
				App:        "sonarr",
				APIVersion: "v3",
			},
			expectedMetricsFile: "expected_metrics_extended.txt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := newTestSonarrServer(t, func(_ http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "/api/")
			})
			assert.NoError(t, err)

			defer ts.Close()

			tt.config.URL = ts.URL
			tt.config.APIKey = fixtures.APIKey

			cl, err := client.NewClient(tt.config)
			assert.NoError(t, err)
			collector := NewSonarrCollector(cl, tt.config)
			assert.NoError(t, err)

			b, err := os.ReadFile(sonarrTestFixturesPath + tt.expectedMetricsFile)
			assert.NoError(t, err)

			expected := strings.ReplaceAll(string(b), "SOMEURL", ts.URL)
			f := strings.NewReader(expected)

			assert.NotPanics(t, func() {
				err = testutil.CollectAndCompare(collector, f)
			})
			assert.NoError(t, err)
		})
	}
}

func TestSonarrCollect_FailureDoesntPanic(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	config := &config.ArrConfig{
		URL:    ts.URL,
		APIKey: fixtures.APIKey,
	}
	cl, err := client.NewClient(config)
	assert.NoError(t, err)
	collector := NewRadarrCollector(cl, config)

	f := strings.NewReader("")

	assert.NotPanics(t, func() {
		err := testutil.CollectAndCompare(collector, f)
		assert.Error(t, err)
	}, "Collecting metrics should not panic on failure")
}

// TestSonarrCollect_DisableWantedMetrics proves the wanted endpoints are never
// queried when disabled — their totals force full table counts that can hang
// multi-year instances.
func TestSonarrCollect_DisableWantedMetrics(t *testing.T) {
	ts, err := newTestSonarrServer(t, func(_ http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/wanted/") {
			t.Errorf("wanted endpoint %q must not be queried when disabled", r.URL.Path)
		}
	})
	assert.NoError(t, err)
	defer ts.Close()

	config := &config.ArrConfig{
		App:                  "sonarr",
		APIVersion:           "v3",
		URL:                  ts.URL,
		APIKey:               fixtures.APIKey,
		DisableWantedMetrics: true,
	}
	cl, err := client.NewClient(config)
	assert.NoError(t, err)
	collector := NewSonarrCollector(cl, config)

	assert.GreaterOrEqual(t, testutil.CollectAndCount(collector), 5)
	assert.Equal(t, testutil.CollectAndCount(collector, "sonarr_episode_missing_total", "sonarr_episode_cutoff_unmet_total"), 0,
		"wanted series must be absent when disabled")
	assert.Equal(t, testutil.CollectAndCount(collector, "sonarr_collector_error"), 0)
}
