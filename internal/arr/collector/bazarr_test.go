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
	"slices"
)

const bazarrTestFixturesPath = "../testdata/bazarr/"

func newTestBazarrServer(t *testing.T, fn func(http.ResponseWriter, *http.Request)) (*httptest.Server, error) {
	return fixtures.NewTestServer(t, bazarrTestFixturesPath, fn)
}

func TestBazarrCollect(t *testing.T) {
	ts, err := newTestBazarrServer(t, func(_ http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/")
	})
	assert.NoError(t, err)

	defer ts.Close()

	config := &config.ArrConfig{
		URL:                   ts.URL,
		App:                   "bazarr",
		APIKey:                fixtures.APIKey,
		DisableEpisodeMetrics: true,
		Bazarr: config.BazarrConfig{
			SeriesBatchSize:        1,
			SeriesBatchConcurrency: 1,
		},
	}
	cl, err := client.NewClient(config)
	assert.NoError(t, err)
	collector := NewBazarrCollector(cl, config)
	assert.NoError(t, err)

	b, err := os.ReadFile(bazarrTestFixturesPath + "expected_metrics.txt")
	assert.NoError(t, err)

	expected := strings.ReplaceAll(string(b), "SOMEURL", ts.URL)
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
		"bazarr_subtitles_score",
		"bazarr_subtitles_unmonitored_total",
		"bazarr_subtitles_wanted_total",
		"bazarr_signalr_connected",
		"bazarr_system_health_issues",
		"bazarr_system_status",
		"bazarr_throttled_providers",
		"exportarr_app_info",
	}

	assert.NotPanics(t, func() {
		err = testutil.CollectAndCompare(collector, f,
			collections...,
		)
	})
	assert.NoError(t, err)

	// TODO: can this become more magic?
	totalLanguages := 1
	totalScores := 1 // the score histogram gathers as a single metric
	totalProviders := 3

	// +1: bazarr_signalr_connected emits one series per upstream app.
	assert.GreaterOrEqual(t, len(collections)+totalLanguages+totalProviders+totalScores+1, testutil.CollectAndCount(collector))
}

func TestBazarrCollect_Concurrency(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/")

		switch r.URL.Path {
		case "/api/series":
			w.WriteHeader(http.StatusOK)
			json, err := os.ReadFile(bazarrTestFixturesPath + "concurrency/series.json")
			assert.NoError(t, err)
			_, err = w.Write(json)
			assert.NoError(t, err)

		case "/api/episodes":
			seriesIDs := r.URL.Query()["seriesid[]"]
			assert.Len(t, seriesIDs, 2)

			if slices.Contains(seriesIDs, "944") && slices.Contains(seriesIDs, "945") {
				w.WriteHeader(http.StatusOK)
				json, err := os.ReadFile(bazarrTestFixturesPath + "concurrency/episodes944_945.json")
				assert.NoError(t, err)
				_, err = w.Write(json)
				assert.NoError(t, err)
			} else if slices.Contains(seriesIDs, "946") && slices.Contains(seriesIDs, "947") {
				w.WriteHeader(http.StatusOK)
				json, err := os.ReadFile(bazarrTestFixturesPath + "concurrency/episodes946_947.json")
				assert.NoError(t, err)
				_, err = w.Write(json)
				assert.NoError(t, err)
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}

		default:
			ts2, err := newTestBazarrServer(t, func(_ http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "/api/")
			})
			assert.NoError(t, err)
			ts2.Config.Handler.ServeHTTP(w, r)
		}
	}))

	defer ts.Close()

	config := &config.ArrConfig{
		URL:    ts.URL,
		App:    "bazarr",
		APIKey: fixtures.APIKey,
		Bazarr: config.BazarrConfig{
			SeriesBatchSize:        2,
			SeriesBatchConcurrency: 2,
		},
	}
	cl, err := client.NewClient(config)
	assert.NoError(t, err)
	collector := NewBazarrCollector(cl, config)

	b, err := os.ReadFile(bazarrTestFixturesPath + "concurrency/expected_metrics.txt")
	assert.NoError(t, err)

	expected := strings.ReplaceAll(string(b), "SOMEURL", ts.URL)
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
		"bazarr_subtitles_score",
		"bazarr_subtitles_unmonitored_total",
		"bazarr_subtitles_wanted_total",
		"bazarr_signalr_connected",
		"bazarr_system_health_issues",
		"bazarr_system_status",
		"bazarr_throttled_providers",
		"exportarr_app_info",
	}

	assert.NotPanics(t, func() {
		err = testutil.CollectAndCompare(collector, f,
			collections...,
		)
	})
	assert.NoError(t, err)

	// TODO: can this become more magic?
	totalLanguages := 1
	totalScores := 1 // the score histogram gathers as a single metric
	totalProviders := 3

	// +1: bazarr_signalr_connected emits one series per upstream app.
	assert.GreaterOrEqual(t, len(collections)+totalLanguages+totalProviders+totalScores+1, testutil.CollectAndCount(collector))
}

func TestBazarrCollect_FailureDoesntPanic(t *testing.T) {

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
	collector := NewBazarrCollector(cl, config)

	f := strings.NewReader("")

	assert.NotPanics(t, func() {
		err := testutil.CollectAndCompare(collector, f)
		assert.Error(t, err)
	}, "Collecting metrics should not panic on failure")
}

// TestBazarrCollect_SkipsOverlappingScrapes pins
// https://github.com/onedr0p/exportarr/issues/380: while one collection is
// still running against a slow instance, a second scrape must skip (emitting
// only the error gauge) instead of stacking more load onto the app.
func TestBazarrCollect_SkipsOverlappingScrapes(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[],"total":0}`))
	}))
	defer ts.Close()

	config := &config.ArrConfig{
		URL:    ts.URL,
		App:    "bazarr",
		APIKey: fixtures.APIKey,
		Bazarr: config.BazarrConfig{
			SeriesBatchSize:        1,
			SeriesBatchConcurrency: 1,
		},
	}
	cl, err := client.NewClient(config)
	assert.NoError(t, err)
	collector := NewBazarrCollector(cl, config)

	done := make(chan int, 1)
	go func() { done <- testutil.CollectAndCount(collector) }()
	<-started // the first collection is now blocked inside the server

	// The overlapping scrape must return immediately with just the error gauge.
	assert.Equal(t, testutil.CollectAndCount(collector), 1)

	close(release) // let the first collection finish
	<-done
}

// TestBazarrCollect_NoSeries pins https://github.com/onedr0p/exportarr/issues/244:
// a bazarr instance with no series must scrape cleanly — in particular the
// episodes endpoint must never be called with an empty series list (bazarr
// answers that with a 404).
func TestBazarrCollect_NoSeries(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/episodes") && !strings.Contains(r.URL.Path, "history") {
			t.Errorf("episodes endpoint must not be queried when there are no series")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		switch {
		case strings.HasSuffix(r.URL.Path, "/badges"):
			_, _ = w.Write([]byte(`{"episodes":0,"providers":0,"sonarr_signalr":"LIVE","radarr_signalr":"LIVE"}`))
		default:
			_, _ = w.Write([]byte(`{"data":[],"total":0}`))
		}
	}))
	defer ts.Close()

	config := &config.ArrConfig{
		URL:    ts.URL,
		App:    "bazarr",
		APIKey: fixtures.APIKey,
		Bazarr: config.BazarrConfig{
			SeriesBatchSize:        300,
			SeriesBatchConcurrency: 10,
		},
	}
	cl, err := client.NewClient(config)
	assert.NoError(t, err)
	collector := NewBazarrCollector(cl, config)

	count := testutil.CollectAndCount(collector)
	assert.GreaterOrEqual(t, count, 20, "an empty bazarr must still emit its metric families")
	assert.Equal(t, testutil.CollectAndCount(collector, "bazarr_collector_error"), 0,
		"an empty bazarr must not raise the error gauge")
}
