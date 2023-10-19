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

const radarr_test_fixtures_path = "../test_fixtures/radarr/"

func newTestRadarrServer(t *testing.T, fn func(http.ResponseWriter, *http.Request)) (*httptest.Server, error) {
	return test_util.NewTestServer(t, radarr_test_fixtures_path, fn)
}

func TestRadarrCollect(t *testing.T) {
	require := require.New(t)
	ts, err := newTestRadarrServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Contains(r.URL.Path, "/api/")
	})
	require.NoError(err)

	defer ts.Close()

	config := &config.ArrConfig{
		URL:        ts.URL,
		App:        "radarr",
		ApiKey:     test_util.API_KEY,
		ApiVersion: "v3",
	}
	collector := NewRadarrCollector(config)
	require.NoError(err)

	b, err := os.ReadFile(radarr_test_fixtures_path + "expected_metrics.txt")
	require.NoError(err)

	expected := strings.Replace(string(b), "SOMEURL", ts.URL, -1)
	f := strings.NewReader(expected)

	require.NotPanics(func() {
		err = testutil.CollectAndCompare(collector, f)
	})
	require.NoError(err)
}

func TestRadarrCollect_FailureDoesntPanic(t *testing.T) {
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
