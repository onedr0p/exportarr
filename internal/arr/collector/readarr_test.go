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

const readarr_test_fixtures_path = "../test_fixtures/readarr/"

func newTestReadarrServer(t *testing.T, fn func(http.ResponseWriter, *http.Request)) (*httptest.Server, error) {
	return test_util.NewTestServer(t, readarr_test_fixtures_path, fn)
}

func TestReadarrCollect(t *testing.T) {
	require := require.New(t)
	ts, err := newTestReadarrServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Contains(r.URL.Path, "/api/")
	})
	require.NoError(err)

	defer ts.Close()

	config := &config.ArrConfig{
		URL:        ts.URL,
		App:        "radarr",
		ApiKey:     test_util.API_KEY,
		ApiVersion: "v1",
	}
	collector := NewReadarrCollector(config)
	require.NoError(err)

	b, err := os.ReadFile(readarr_test_fixtures_path + "expected_metrics.txt")
	require.NoError(err)

	expected := strings.ReplaceAll(string(b), "SOMEURL", ts.URL)
	f := strings.NewReader(expected)

	require.NotPanics(func() {
		err = testutil.CollectAndCompare(collector, f)
	})
	require.NoError(err)
}

func TestReadarrCollect_FailureDoesntPanic(t *testing.T) {
	require := require.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	config := &config.ArrConfig{
		URL:    ts.URL,
		ApiKey: test_util.API_KEY,
	}
	collector := NewReadarrCollector(config)

	f := strings.NewReader("")

	require.NotPanics(func() {
		err := testutil.CollectAndCompare(collector, f)
		require.Error(err)
	}, "Collecting metrics should not panic on failure")
}
