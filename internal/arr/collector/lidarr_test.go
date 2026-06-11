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

const lidarrTestFixturesPath = "../testdata/lidarr/"

func newTestLidarrServer(t *testing.T, fn func(http.ResponseWriter, *http.Request)) (*httptest.Server, error) {
	return fixtures.NewTestServer(t, lidarrTestFixturesPath, fn)
}

func TestLidarrCollect(t *testing.T) {
	tests := []struct {
		name                string
		config              *config.ArrConfig
		expectedMetricsFile string
	}{
		{
			name: "basic",
			config: &config.ArrConfig{
				App:                   "lidarr",
				APIVersion:            "v3",
				DisableQualityMetrics: true,
				DisableAlbumMetrics:   true,
			},
			expectedMetricsFile: "expected_metrics.txt",
		},
		{
			name: "default_collects_everything",
			config: &config.ArrConfig{
				App:        "lidarr",
				APIVersion: "v3",
			},
			expectedMetricsFile: "expected_metrics_extended.txt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := newTestLidarrServer(t, func(_ http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "/api/")
			})
			assert.NoError(t, err)

			defer ts.Close()

			tt.config.URL = ts.URL
			tt.config.APIKey = fixtures.APIKey

			cl, err := client.NewClient(tt.config)
			assert.NoError(t, err)
			collector := NewLidarrCollector(cl, tt.config)
			assert.NoError(t, err)

			b, err := os.ReadFile(lidarrTestFixturesPath + tt.expectedMetricsFile)
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

func TestLidarrCollect_FailureDoesntPanic(t *testing.T) {

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
