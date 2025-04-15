package collector

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/test_util"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestRootFolderCollect(t *testing.T) {
	var tests = []struct {
		name   string
		config *config.ArrConfig
		path   string
	}{
		{
			name: "radarr",
			config: &config.ArrConfig{
				App:        "radarr",
				ApiVersion: "v3",
			},
			path: "/api/v3/rootfolder",
		},
		{
			name: "sonarr",
			config: &config.ArrConfig{
				App:        "sonarr",
				ApiVersion: "v3",
			},
			path: "/api/v3/rootfolder",
		},
		{
			name: "lidarr",
			config: &config.ArrConfig{
				App:        "lidarr",
				ApiVersion: "v1",
			},
			path: "/api/v1/rootfolder",
		},
		{
			name: "readarr",
			config: &config.ArrConfig{
				App:        "readarr",
				ApiVersion: "v1",
			},
			path: "/api/v1/rootfolder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			ts, err := test_util.NewTestSharedServer(t, func(w http.ResponseWriter, r *http.Request) {
				require.Contains(r.URL.Path, tt.path)
			})
			require.NoError(err)

			defer ts.Close()

			tt.config.URL = ts.URL
			tt.config.ApiKey = test_util.API_KEY

			collector := NewRootFolderCollector(tt.config)

			b, err := os.ReadFile(test_util.COMMON_FIXTURES_PATH + "expected_rootfolder_metrics.txt")
			require.NoError(err)

			expected := strings.ReplaceAll(string(b), "SOMEURL", ts.URL)
			expected = strings.ReplaceAll(expected, "APP", tt.config.App)

			f := strings.NewReader(expected)

			require.NotPanics(func() {
				err = testutil.CollectAndCompare(collector, f)
			})
			require.NoError(err)
		})
	}
}

func TestRootFolderCollect_FailureDoesntPanic(t *testing.T) {
	require := require.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	config := &config.ArrConfig{
		URL:    ts.URL,
		ApiKey: test_util.API_KEY,
	}
	collector := NewRootFolderCollector(config)

	f := strings.NewReader("")

	require.NotPanics(func() {
		err := testutil.CollectAndCompare(collector, f)
		require.Error(err)
	}, "Collecting metrics should not panic on failure")
}
