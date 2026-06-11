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

const radarrTestFixturesPath = "../testdata/radarr/"

func newTestRadarrServer(t *testing.T, fn func(http.ResponseWriter, *http.Request)) (*httptest.Server, error) {
	return fixtures.NewTestServer(t, radarrTestFixturesPath, fn)
}

func TestRadarrCollect(t *testing.T) {
	ts, err := newTestRadarrServer(t, func(_ http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/")
	})
	assert.NoError(t, err)

	defer ts.Close()

	config := &config.ArrConfig{
		URL:        ts.URL,
		App:        "radarr",
		APIKey:     fixtures.APIKey,
		APIVersion: "v3",
	}
	cl, err := client.NewClient(config)
	assert.NoError(t, err)
	collector := NewRadarrCollector(cl, config)
	assert.NoError(t, err)

	b, err := os.ReadFile(radarrTestFixturesPath + "expected_metrics.txt")
	assert.NoError(t, err)

	expected := strings.ReplaceAll(string(b), "SOMEURL", ts.URL)
	f := strings.NewReader(expected)

	assert.NotPanics(t, func() {
		err = testutil.CollectAndCompare(collector, f)
	})
	assert.NoError(t, err)
}

func TestRadarrCollect_FailureDoesntPanic(t *testing.T) {

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
