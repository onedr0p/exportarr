package collector

import (
	"github.com/onedr0p/exportarr/internal/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/onedr0p/exportarr/internal/sabnzbd/config"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

const testAPIKey = "abcdef0123456789abcdef0123456789"

func newTestServer(t *testing.T, fn func(http.ResponseWriter, *http.Request)) (*httptest.Server, error) {
	queue, err := os.ReadFile("../testdata/queue.json")
	assert.NoError(t, err)
	serverStats, err := os.ReadFile("../testdata/server_stats.json")
	assert.NoError(t, err)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fn(w, r)
		assert.NotEmpty(t, r.URL.Query().Get("mode"))
		switch r.URL.Query().Get("mode") {
		case "queue":
			w.WriteHeader(http.StatusOK)
			_, err := w.Write(queue)
			assert.NoError(t, err)
		case "server_stats":
			w.WriteHeader(http.StatusOK)
			_, err := w.Write(serverStats)
			assert.NoError(t, err)
		}
	})), nil
}

func TestCollect(t *testing.T) {
	ts, err := newTestServer(t, func(_ http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api")
		assert.Equal(t, r.URL.Query().Get("apikey"), testAPIKey)
		assert.Equal(t, r.URL.Query().Get("output"), "json")
	})
	assert.NoError(t, err)

	defer ts.Close()

	config := &config.SabnzbdConfig{
		URL:    ts.URL,
		APIKey: testAPIKey,
	}
	collector, err := NewSabnzbdCollector(config)
	assert.NoError(t, err)

	b, err := os.ReadFile("../testdata/expected_metrics.txt")
	assert.NoError(t, err)

	expected := strings.ReplaceAll(string(b), "http://127.0.0.1:39965", ts.URL)
	f := strings.NewReader(expected)

	assert.NotPanics(t, func() {
		err = testutil.CollectAndCompare(collector, f,
			"sabnzbd_downloaded_bytes",
			"sabnzbd_server_downloaded_bytes",
			"sabnzbd_server_articles_total",
			"sabnzbd_server_articles_success",
			"sabnzbd_info",
			"sabnzbd_paused",
			"sabnzbd_paused_all",
			"sabnzbd_pause_duration_seconds",
			"sabnzbd_disk_used_bytes",
			"sabnzbd_disk_total_bytes",
			"sabnzbd_remaining_quota_bytes",
			"sabnzbd_quota_bytes",
			"sabnzbd_article_cache_articles",
			"sabnzbd_article_cache_bytes",
			"sabnzbd_speed_bps",
			"sabnzbd_speed_limit_bps",
			"sabnzbd_speed_limit_percent",
			"sabnzbd_remaining_bytes",
			"sabnzbd_total_bytes",
			"sabnzbd_queue_size",
			"sabnzbd_status",
			"sabnzbd_time_estimate_seconds",
			"sabnzbd_queue_length",
			"sabnzbd_warnings",
		)
	})
	assert.NoError(t, err)

	assert.GreaterOrEqual(t, 31, testutil.CollectAndCount(collector))
}

func TestCollect_FailureDoesntPanic(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	config := &config.SabnzbdConfig{
		URL:    ts.URL,
		APIKey: testAPIKey,
	}
	collector, err := NewSabnzbdCollector(config)
	assert.NoError(t, err)

	f := strings.NewReader("")

	assert.NotPanics(t, func() {
		err = testutil.CollectAndCompare(collector, f)
		assert.Error(t, err)
	}, "Collecting metrics should not panic on failure")
	assert.Error(t, err)
}
