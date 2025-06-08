package collector

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/shamelin/exportarr/internal/arr/config"
	"github.com/shamelin/exportarr/internal/test_util"
	"github.com/stretchr/testify/require"
)

const lidarr_test_fixtures_path = "../test_fixtures/lidarr/"

func newTestLidarrServer(t *testing.T, fn func(http.ResponseWriter, *http.Request)) (*httptest.Server, error) {
	return test_util.NewTestServer(t, lidarr_test_fixtures_path, fn)
}

func TestLidarrCollect(t *testing.T) {
	tests := []struct {
		name                  string
		config                *config.ArrConfig
		expected_metrics_file string
	}{
		{
			name: "basic",
			config: &config.ArrConfig{
				App:        "lidarr",
				ApiVersion: "v3",
			},
			expected_metrics_file: "expected_metrics.txt",
		},
		{
			name: "additional_metrics",
			config: &config.ArrConfig{
				App:                     "lidarr",
				ApiVersion:              "v3",
				EnableAdditionalMetrics: true,
			},
			expected_metrics_file: "expected_metrics_extended.txt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			ts, err := newTestLidarrServer(t, func(w http.ResponseWriter, r *http.Request) {
				require.Contains(r.URL.Path, "/api/")
			})
			require.NoError(err)

			defer ts.Close()

			tt.config.URL = ts.URL
			tt.config.ApiKey = test_util.API_KEY

			collector := NewLidarrCollector(tt.config)
			require.NoError(err)

			b, err := os.ReadFile(lidarr_test_fixtures_path + tt.expected_metrics_file)
			require.NoError(err)

			expected := strings.ReplaceAll(string(b), "SOMEURL", ts.URL)
			f := strings.NewReader(expected)

			require.NotPanics(func() {
				err = testutil.CollectAndCompare(collector, f)
			})
			require.NoError(err)
		})
	}
}

func TestLidarrCollect_FailureDoesntPanic(t *testing.T) {
	require := require.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	config := &config.ArrConfig{
		URL:    ts.URL,
		ApiKey: test_util.API_KEY,
	}
	collector := NewRadarrCollector(config)

	f := strings.NewReader("")

	require.NotPanics(func() {
		err := testutil.CollectAndCompare(collector, f)
		require.Error(err)
	}, "Collecting metrics should not panic on failure")
}
