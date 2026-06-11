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

func TestHistoryCollect(t *testing.T) {
	var tests = []struct {
		name   string
		config *config.ArrConfig
		path   string
	}{
		{
			name: "radarr",
			config: &config.ArrConfig{
				App:        "radarr",
				APIVersion: "v3",
			},
			path: "/api/v3/history",
		},
		{
			name: "sonarr",
			config: &config.ArrConfig{
				App:        "sonarr",
				APIVersion: "v3",
			},
			path: "/api/v3/history",
		},
		{
			name: "lidarr",
			config: &config.ArrConfig{
				App:        "lidarr",
				APIVersion: "v1",
			},
			path: "/api/v1/history",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := fixtures.NewTestSharedServer(t, func(_ http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, tt.path)
				assert.Equal(t, r.URL.Query().Get("pageSize"), "1")
			})
			assert.NoError(t, err)

			defer ts.Close()

			tt.config.URL = ts.URL
			tt.config.APIKey = fixtures.APIKey

			cl, err := client.NewClient(tt.config)
			assert.NoError(t, err)
			collector := NewHistoryCollector(cl, tt.config)

			b, err := os.ReadFile(fixtures.CommonFixturesPath + "expected_history_metrics.txt")
			assert.NoError(t, err)

			expected := strings.ReplaceAll(string(b), "SOMEURL", ts.URL)
			expected = strings.ReplaceAll(expected, "APP", tt.config.App)

			f := strings.NewReader(expected)

			assert.NotPanics(t, func() {
				err = testutil.CollectAndCompare(collector, f)
			})
			assert.NoError(t, err)
		})
	}
}

func TestHistoryCollect_FailureDoesntPanic(t *testing.T) {

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
	collector := NewHistoryCollector(cl, config)

	f := strings.NewReader("")

	assert.NotPanics(t, func() {
		err := testutil.CollectAndCompare(collector, f)
		assert.Error(t, err)
	}, "Collecting metrics should not panic on failure")
}
