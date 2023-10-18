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

const bazarr_test_fixtures_path = "../test_fixtures/bazarr/"

func newTestBazarrServer(t *testing.T, fn func(http.ResponseWriter, *http.Request)) (*httptest.Server, error) {
	return test_util.NewTestServer(t, bazarr_test_fixtures_path, fn)
}

func TestBazarrCollect(t *testing.T) {
	require := require.New(t)
	ts, err := newTestBazarrServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Contains(r.URL.Path, "/api/")
	})
	require.NoError(err)

	defer ts.Close()

	config := &config.ArrConfig{
		URL:    ts.URL,
		App:    "bazarr",
		ApiKey: test_util.API_KEY,
	}
	collector := NewBazarrCollector(config)
	require.NoError(err)

	b, err := os.ReadFile(bazarr_test_fixtures_path + "expected_metrics.txt")
	require.NoError(err)

	expected := strings.Replace(string(b), "SOMEURL", ts.URL, -1)
	f := strings.NewReader(expected)
	collections := []string{
		"bazarr_episode_subtitles_downloaded_total",
		"bazarr_episode_subtitles_filesize_total",
		"bazarr_episode_subtitles_history_total",
		"bazarr_episode_subtitles_missing_total",
		"bazarr_episode_subtitles_monitored_total",
		"bazarr_episode_subtitles_unmonitored_total",
		"bazarr_episode_subtitles_wanted_total",
		"bazarr_movie_subtitles_downloaded_total",
		"bazarr_movie_subtitles_filesize_total",
		"bazarr_movie_subtitles_history_total",
		"bazarr_movie_subtitles_missing_total",
		"bazarr_movie_subtitles_monitored_total",
		"bazarr_movie_subtitles_unmonitored_total",
		"bazarr_movie_subtitles_wanted_total",
		"bazarr_scrape_duration_seconds",
		"bazarr_scrape_requests_total",
		"bazarr_subtitles_downloaded_total",
		"bazarr_subtitles_filesize_total",
		"bazarr_subtitles_history_total",
		"bazarr_subtitles_language_total",
		"bazarr_subtitles_missing_total",
		"bazarr_subtitles_monitored_total",
		"bazarr_subtitles_provider_total",
		"bazarr_subtitles_score_total",
		"bazarr_subtitles_unmonitored_total",
		"bazarr_subtitles_wanted_total",
		"bazarr_system_health_issues",
		"bazarr_system_status",
		"exportarr_app_info",
	}

	require.NotPanics(func() {
		err = testutil.CollectAndCompare(collector, f,
			collections...,
		)
	})
	require.NoError(err)

	// TODO: can this become more magic?
	totalLanguages := 1
	totalScores := 15
	totalProviders := 3

	require.GreaterOrEqual(len(collections)+totalLanguages+totalProviders+totalScores, testutil.CollectAndCount(collector))
}

func TestBazarrCollect_FailureDoesntPanic(t *testing.T) {
	require := require.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	config := &config.ArrConfig{
		URL:    ts.URL,
		ApiKey: test_util.API_KEY,
	}
	collector := NewBazarrCollector(config)

	f := strings.NewReader("")

	require.NotPanics(func() {
		err := testutil.CollectAndCompare(collector, f)
		require.Error(err)
	}, "Collecting metrics should not panic on failure")
}
